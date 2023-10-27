ARG BUILD_ENV=dapper

FROM registry.suse.com/bci/bci-base:15.5.36.5.34 AS base

FROM base AS copy_dapper
ONBUILD ARG OS
ONBUILD COPY bin/rancher-system-agent-${OS} /usr/bin/rancher-system-agent

FROM base AS copy_buildx
ONBUILD ARG TARGETARCH
ONBUILD COPY bin/rancher-system-agent-${TARGETARCH} /usr/bin/rancher-system-agent

FROM copy_${BUILD_ENV}
CMD ["rancher-system-agent"]
