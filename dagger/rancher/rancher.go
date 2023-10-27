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

func (k *K8sInstance) InstallRancher() error {
	// create k3s service container
	k3s := k.client.Pipeline("k3s init").Container().
		From("rancher/k3s").
		WithMountedCache("/etc/rancher/k3s", k.configCache).
		WithNewFile("/cgroupv2_entrypoint.sh", dagger.ContainerWithNewFileOpts{
			Contents: `
#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

#########################################################################################################################################
# DISCLAIMER																																																														#
# Copied from https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/hack/dind#L28-L37															#
# Permission granted by Akihiro Suda <akihiro.suda.cz@hco.ntt.co.jp> (https://github.com/rancher/k3d/issues/493#issuecomment-827405962)	#
# Moby License Apache 2.0: https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/LICENSE														#
#########################################################################################################################################
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
	# move the processes from the root group to the /init group,
  # otherwise writing subtree_control fails with EBUSY.
  mkdir -p /sys/fs/cgroup/init
  busybox xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
  # enable controllers
  sed -e 's/ / +/g' -e 's/^/+/' <"/sys/fs/cgroup/cgroup.controllers" >"/sys/fs/cgroup/cgroup.subtree_control"
fi

exec "$@"
`[1:],
			Permissions: 0755,
			Owner:       "root",
		}).
		WithNewFile("/k3s.sh", dagger.ContainerWithNewFileOpts{
			Contents: `
#!/bin/sh

set -o errexit
set -o nounset
set -o pipefail

while true; do
  bind_address=$(ip -4 -brief addr show scope global up | awk '{print $3}' | cut -d/ -f1)
	if [ "$bind_address" != "linkdown" ]; then
	  break
	fi
	sleep 0.5
done
exec /bin/k3s server --snapshotter=native --bind-address=$bind_address \
	--disable=metrics-server --disable=traefik --disable=cloud-controller \
	--write-kubeconfig-mode=644
`[1:],
			Permissions: 0755,
			Owner:       "root",
		}).
		WithEntrypoint([]string{"/cgroupv2_entrypoint.sh"}).
		WithEnvVariable("CACHE", time.Now().String()).
		WithExec([]string{"/k3s.sh"}, dagger.ContainerWithExecOpts{InsecureRootCapabilities: true}).
		WithExposedPort(6443)

	val, err := k.client.Container().
		From("bitnami/kubectl").
		WithEnvVariable("CACHE", time.Now().String()).
		WithMountedCache("/cache/k3s", k.configCache).
		WithServiceBinding("k3s", k3s).
		WithEnvVariable("KUBECONFIG", "/cache/k3s/k3s.yaml").
		WithEntrypoint([]string{"sh", "-c"}).
		WithExec([]string{`while [ "$(kubectl get --raw /healthz 2>/dev/null)" != "ok" ]; do sleep 0.5; done`}).
		Stdout(k.ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(val)

	return nil
}
