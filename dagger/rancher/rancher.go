package rancher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dagger.io/dagger"
)

func NewK8sInstance(ctx context.Context, client *dagger.Client) *K8sInstance {
	return &K8sInstance{
		ctx:         ctx,
		client:      client,
		container:   nil,
		configCache: client.CacheVolume("k3s_config"),
	}
}

type K8sInstance struct {
	ctx         context.Context
	client      *dagger.Client
	container   *dagger.Container
	configCache *dagger.CacheVolume
}

func (k *K8sInstance) Start() error {
	// create k3s service container
	k3s := k.client.Pipeline("k3s init").Container().
		From("rancher/k3s").
		WithMountedCache("/etc/rancher/k3s", k.configCache).
		WithMountedTemp("/etc/lib/cni").
		WithMountedTemp("/var/lib/kubelet").
		WithMountedTemp("/var/lib/rancher/k3s").
		WithMountedTemp("/var/log").
		WithEntrypoint([]string{"sh", "-c"}).
		WithExec([]string{"k3s server --bind-address $(ip route | grep src | awk '{print $NF}') --disable traefik --disable metrics-server"}, dagger.ContainerWithExecOpts{InsecureRootCapabilities: true}).
		WithExposedPort(6443)

	k.container = k.client.Container().
		From("dtzar/helm-kubectl:3.12").
		WithMountedCache("/cache/k3s", k.configCache).
		//WithDirectory("/host", k.client.Host().Directory(parentDir)).
		WithServiceBinding("k3s", k3s).
		WithEnvVariable("CACHE", time.Now().String()).
		WithUser("root").
		WithExec([]string{"mkdir", "-p", "/.kube"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"mkdir", "-p", "/.config/helm"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"mkdir", "-p", "/.cache/helm"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"cp", "/cache/k3s/k3s.yaml", "/.kube/config"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"cat", "/.kube/config"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"chown", "1001:0", "/.kube/config"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"chown", "1001:0", "/.config/helm"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithExec([]string{"chown", "1001:0", "/.cache/helm"}, dagger.ContainerWithExecOpts{SkipEntrypoint: true}).
		WithUser("1001").
		WithEntrypoint([]string{"sh", "-c"})

	if err := k.waitForNodes(); err != nil {
		return fmt.Errorf("failed to start k8s: %v", err)
	}
	return nil
}

func (k *K8sInstance) kubectl(command string) (string, error) {
	return k.exec("kubectl", fmt.Sprintf("kubectl %v", command))
}

func (k *K8sInstance) helm(command string) (string, error) {
	return k.exec("helm", fmt.Sprintf("helm %v", command))
}

//func (k *K8sInstance) exec(name, command string) (string, error) {
//	return k.container.
//		Pipeline(name).
//		Pipeline(command).
//		WithEnvVariable("CACHE", time.Now().String()).
//		WithExec([]string{command}).
//		Stdout(k.ctx)
//}

func (k *K8sInstance) exec(name, command string) (string, error) {
	return k.container.Pipeline(name).Pipeline(command).
		WithEnvVariable("CACHE", time.Now().String()).
		WithExec([]string{command}).
		Stdout(k.ctx)
}

func (k *K8sInstance) waitForNodes() (err error) {
	maxRetries := 5
	retryBackoff := 5 * time.Second
	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryBackoff)
		kubectlGetNodes, err := k.kubectl("get nodes -o wide")
		if err != nil {
			fmt.Println(fmt.Errorf("could not fetch nodes: %v", err))
			continue
		}
		if strings.Contains(kubectlGetNodes, "Ready") {
			return nil
		}
		fmt.Println("waiting for k8s to start:", kubectlGetNodes)
	}
	return fmt.Errorf("k8s took too long to start")
}

func (k *K8sInstance) InstallRancher() {
	pods, err := k.kubectl("get pods -A -o wide")
	if err != nil {
		panic(err)
	}
	fmt.Println(pods)

	services, err := k.kubectl("get services -A -o wide")
	if err != nil {
		panic(err)
	}
	fmt.Println(services)

	_, err = k.helm("repo add rancher-latest https://releases.rancher.com/server-charts/latest")
	if err != nil {
		panic(err)
	}

	_, err = k.helm("repo update rancher-latest")
	if err != nil {
		panic(err)
	}

	_, err = k.kubectl("apply -f https://github.com/jetstack/cert-manager/releases/download/v1.5.4/cert-manager.yaml")
	if err != nil {
		panic(err)
	}

	_, err = k.kubectl(`upgrade rancher rancher-latest/rancher \
  --devel \
  --install --wait \
  --create-namespace \
  --namespace cattle-system \
  --set hostname="172.18.0.1.sslip.io" \
  --set bootstrapPassword=admin \
  --set "extraEnv[0].name=CATTLE_SERVER_URL" \
  --set "extraEnv[0].value=https://172.18.0.1.sslip.io" \
  --set "extraEnv[1].name=CATTLE_BOOTSTRAP_PASSWORD" \
  --set "extraEnv[1].value=rancherpassword"`)
	if err != nil {
		panic(err)
	}

	_, err = k.kubectl("-n cattle-system rollout status deploy/rancher")
	if err != nil {
		panic(err)
	}
}
