#!/bin/bash

set -x

#########################################################################################################################################
# DISCLAIMER                                                                                                                            #
# Copied from https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/hack/dind#L28-L37                              #
# Permission granted by Akihiro Suda <akihiro.suda.cz@hco.ntt.co.jp> (https://github.com/rancher/k3d/issues/493#issuecomment-827405962) #
# Moby License Apache 2.0: https://github.com/moby/moby/blob/ed89041433a031cafc0a0f19cfe573c31688d377/LICENSE                           #
#########################################################################################################################################
# only run this if rancher is not running in kubernetes cluster
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
  # move the processes from the root group to the /init group,
  # otherwise writing subtree_control fails with EBUSY.
  mkdir -p /sys/fs/cgroup/init
  xargs -rn1 < /sys/fs/cgroup/cgroup.procs > /sys/fs/cgroup/init/cgroup.procs || :
  # enable controllers
  sed -e 's/ / +/g' -e 's/^/+/' <"/sys/fs/cgroup/cgroup.controllers" >"/sys/fs/cgroup/cgroup.subtree_control"
fi

#hostname

curl -sfL https://github.com/k3s-io/k3s/releases/download/v1.27.7%2Bk3s1/k3s -o /usr/local/bin/k3s
chmod +x /usr/local/bin/k3s

k3s server --cluster-init \
  --disable=traefik,servicelb,metrics-server,local-storage \
  --node-name=local-node \
  --log=./k3s.log &

#cat k3s.log