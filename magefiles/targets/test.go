package targets

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Test mg.Namespace

// All runs all tests.
func (Test) All(ctx context.Context) {
	mg.CtxDeps(ctx, Test.Unit)
	mg.CtxDeps(ctx, Test.E2E)
}

func (Test) Unit(ctx context.Context) error {
	return sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, mg.GoCmd(), "test", "-race", "-short", "./...")
}

func (Test) Clean(ctx context.Context) error {
	err := sh.RunV("k3d", "cluster", "delete", "system-agent-default")
	if err != nil {
		return err
	}
	return nil
}

func (Test) E2E(ctx context.Context) error {
	// build images
	mg.SerialCtxDeps(ctx, Package.All)
	mg.SerialCtxDeps(ctx, Test.Clean)

	// create temp dir
	tmp, err := sh.Output("mktemp", "-d")
	if err != nil {
		return err
	}

	branch := "release/v2.8"

	// checkout rancher
	err = sh.RunV("git", "clone",
		"--depth", "1",
		"-b", branch,
		"https://github.com/rancher/rancher.git",
		tmp)
	if err != nil {
		return err
	}

	//	k3dconfig := `# k3d configuration file, saved as e.g. /home/me/myk3dcluster.yaml
	//apiVersion: k3d.io/v1alpha4 # this will change in the future as we make everything more stable
	//kind: Simple # internally, we also have a Cluster config, which is not yet available externally
	//kubeAPI:
	//  host: "host.k3d.internal"
	//  hostIP: "127.0.0.1"
	//  hostPort: "6443"
	//`
	//err = sh.RunV(fmt.Sprintf(`echo "%s" > ./k3d-config.yaml`, k3dconfig))

	// spin up k3d
	err = sh.RunV("k3d", "cluster", "create",
		"--kubeconfig-switch-context",
		"--config", "k3dconfig.yaml",
		//"-p", `80:80@server:0:direct`,
		//"-p", `443:443@server:0:direct`,
		//"--api-port", "127.0.0.1:6443",
		//"--k3s-arg", `'--kubelet-arg=eviction-hard=imagefs.available<1%,nodefs.available<1%@agent:*'`,
		//"--k3s-arg", `'--kubelet-arg=eviction-minimum-reclaim=imagefs.available=1%,nodefs.available=1%@agent:*'`,
		//"--agents", "1",
		//"--network", "nw01",
		"--image", "docker.io/rancher/k3s:v1.27.5-k3s1",
		"--wait",
		"system-agent-default")
	if err != nil {
		return err
	}

	// todo: only if containerized
	sh.RunV("docker", "network", "connect", "k3d-system-agent-default", os.Getenv("HOSTNAME")) //HOSTNAME is dapper container ID

	maxRetries := 5
	retryBackoff := 5 * time.Second
	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryBackoff)
		kubectlGetNodes, err := sh.Output("kubectl", "get", "nodes", "-o", "wide")
		if err != nil {
			fmt.Println(fmt.Errorf("could not fetch nodes: %v", err))
			continue
		}
		if strings.Contains(kubectlGetNodes, "Ready") {
			break
		}
		fmt.Println("waiting for k8s to start:", kubectlGetNodes)
		if i == maxRetries {
			return fmt.Errorf("k8s took too long to start")
		}
	}

	err = sh.RunV("kubectl", "apply", "-f", "https://github.com/jetstack/cert-manager/releases/download/v1.5.4/cert-manager.yaml")
	if err != nil {
		return err
	}

	err = sh.RunV("kubectl", "wait", "--for=condition=Available", "deployment", "--timeout=2m", "-n", "cert-manager", "--all")
	if err != nil {
		return err
	}

	// install rancher helm chart
	err = sh.RunV("helm", "repo", "add", "rancher-latest", "https://releases.rancher.com/server-charts/latest")
	if err != nil {
		return err
	}
	err = sh.RunV("helm", "repo", "update", "rancher-latest")
	if err != nil {
		return err
	}

	rancherVersion := "v2.8-head"

	err = sh.RunV("helm",
		"upgrade", "rancher", "rancher-latest/rancher",
		"--version", rancherVersion,
		"--devel",
		"--install",
		"--wait",
		"--namespace", "cattle-system",
		"--create-namespace",
		"--set", "replicas=1",
		"--set", "hostname=k3d-system-agent-default-server-0",
		"--set", "bootstrapPassword=admin",
		"--set", "extraEnv[0].name=CATTLE_SERVER_URL", //
		"--set", "extraEnv[0].value=https://k3d-system-agent-default-server-0",
		"--set", "extraEnv[1].name=CATTLE_BOOTSTRAP_PASSWORD", //
		"--set", "extraEnv[1].value=rancherpassword")
	if err != nil {
		return err
	}

	// wait for rollout
	err = sh.RunV("kubectl",
		"-n", "cattle-system",
		"rollout", "status", "deploy/rancher")
	if err != nil {
		return err
	}

	// run specific rancher e2e tests with custom images
	err = sh.RunWithV(map[string]string{"TMP_DIR": tmp}, "./scripts/e2e.sh")
	if err != nil {
		return err
	}
	return nil
}

// Cover runs all tests with coverage analysis.
func (Test) Cover() error {
	err := sh.RunWithV(map[string]string{
		"CGO_ENABLED": "1",
	}, mg.GoCmd(), "test",
		"-race",
		"-cover",
		"-coverprofile=cover.out",
		"-coverpkg=./...",
		"./...",
	)
	if err != nil {
		return err
	}
	fmt.Print("processing coverage report... ")
	defer fmt.Println("done.")
	return filterCoverage("cover.out", []string{
		"**/*.pb.go",
		"**/*.pb*.go",
		"**/zz_*.go",
	})
}

func filterCoverage(report string, patterns []string) error {
	f, err := os.Open(report)
	if err != nil {
		return err
	}
	defer f.Close()

	tempFile := fmt.Sprintf(".%s.tmp", report)
	tf, err := os.Create(tempFile)
	if err != nil {
		return err
	}

	patternIndex := 0
	scan := bufio.NewScanner(f)
	scan.Scan() // mode line
	_, _ = tf.WriteString(scan.Text() + "\n")
LINES:
	for scan.Scan() {
		line := scan.Text()
		filename, _, _ := strings.Cut(line, ":")
		var j int
		for i := patternIndex; j < len(patterns); i = (i + 1) % len(patterns) {
			match, _ := doublestar.Match(patterns[i], filename)
			if match {
				continue LINES
			}
			j++
		}
		_, _ = tf.WriteString(line + "\n")
	}
	if err := scan.Err(); err != nil {
		return err
	}
	_ = tf.Close()

	return os.Rename(tempFile, report)
}
