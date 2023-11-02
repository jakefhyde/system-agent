#!/bin/bash

set -ex



export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

if [ -z "${SOME_K8S_VERSION}" ]; then
  if [ -z "${DIST}" ]; then
    export DIST="rke2"
  fi
# Get the last release for $DIST, which is usually the latest version or an experimental version.
# Previously this would use channels, but channels no longer reflect the latest version since
# https://github.com/rancher/rancher/issues/36827 has added appDefaults. We do not use appDefaults
# here for simplicity's sake, as it requires semver parsing & matching. The last release should
# be good enough for our needs.
  export SOME_K8S_VERSION=$(curl -sS https://raw.githubusercontent.com/rancher/kontainer-driver-metadata/dev-v2.8/data/data.json | jq -r ".$DIST.releases[-1].version")
fi

echo "Waiting for Rancher to be healthy"
./scripts/retry --sleep 2 --timeout 300 "curl -sf -o /dev/null http://k3d-system-agent-default-server-0:80/ping"

echo "Waiting up to 5 minutes for rancher-webhook deployment"
./scripts/retry \
  --timeout 300 `# Time out after 300 seconds (5 min)` \
  --sleep 2 `# Sleep for 2 seconds in between attempts` \
  --message-interval 30 `# Print the progress message below every 30 attempts (roughly every minute)` \
  --message "rancher-webhook was not available after {{elapsed}} seconds" `# Print this progress message` \
  "kubectl rollout status -w -n cattle-system deploy/rancher-webhook &>/dev/null"

echo "Waiting up to 5 minutes for rancher-provisioning-capi deployment"
./scripts/retry \
  --timeout 300 `# Time out after 300 seconds (5 min)` \
  --sleep 2 `# Sleep for 2 seconds in between attempts` \
  --message-interval 30 `# Print the progress message below every 30 attempts (roughly every minute)` \
  --message "rancher-provisioning-capi was not available after {{elapsed}} seconds" `# Print this progress message` \
  "kubectl rollout status -w -n cattle-provisioning-capi-system deploy/capi-controller-manager &>/dev/null"

cd "${TMP_DIR}" || exit

if [ -z "${V2PROV_TEST_RUN_REGEX}" ]; then
#  V2PROV_TEST_RUN_REGEX="^Test_(General|Provisioning)_.*$"
  V2PROV_TEST_RUN_REGEX="^Test_Provisioning_Custom_ThreeNode$"
fi

RUNARG="-run ${V2PROV_TEST_RUN_REGEX}"

go test $RUNARG -v -failfast -timeout 60m ./tests/v2prov/tests/... || {
  kubectl logs -n cattle-system -l app=rancher --tail 1000
  sleep 10000
  exit 1
}