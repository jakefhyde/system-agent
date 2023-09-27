ARG BUILD_ENV=dapper

FROM registry.suse.com/bci/bci-base:15.5.36.5.34 AS base
USER 1000

FROM base AS copy_dapper
ONBUILD ARG ARCH
ONBUILD COPY bin/rancher-system-agent-${ARCH} /usr/bin/rancher-system-agent

FROM base AS copy_buildx
ONBUILD ARG TARGETARCH
ONBUILD COPY bin/francher-system-agent-${TARGETARCH} /usr/bin/rancher-system-agent

FROM copy_${BUILD_ENV}
CMD ["rancher-system-agent"]
