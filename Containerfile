# Build the ShiftWise operator with Red Hat UBI and a static runtime.
# Builder: UBI 9 Go Toolset (non-root 1001). Runtime: scratch.

ARG GO_TOOLSET_IMAGE=registry.access.redhat.com/ubi9/go-toolset:1.25
ARG RUNTIME_IMAGE=scratch
ARG VERSION=1.0.0
ARG COMPONENT_IMAGE_TAG=1.0.0

FROM ${GO_TOOLSET_IMAGE} AS builder

ARG VERSION
ARG COMPONENT_IMAGE_TAG
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOTOOLCHAIN=go1.26.9 \
    GOFLAGS="-mod=mod"

WORKDIR /opt/app-root/src

COPY --chown=1001:0 go.mod go.sum ./
RUN go mod download

COPY --chown=1001:0 api/ api/
COPY --chown=1001:0 cmd/ cmd/
COPY --chown=1001:0 internal/ internal/

RUN go build -a -ldflags "-s -w -X github.com/ShiftWise-AI/kubeoptix-operator/internal/version.Version=${VERSION} -X github.com/ShiftWise-AI/kubeoptix-operator/internal/constants.ImageTag=${COMPONENT_IMAGE_TAG}" \
    -o /opt/app-root/src/manager cmd/main.go

FROM ${RUNTIME_IMAGE}

ARG VERSION

LABEL name="shiftwise-operator" \
      version="${VERSION}" \
      summary="ShiftWise AI KubeOptix Operator" \
      description="OpenShift operator that reconciles ShiftWise custom resources into the KubeOptix platform." \
      io.k8s.display-name="ShiftWise Operator" \
      io.k8s.description="Manages ShiftWise custom resources on Red Hat OpenShift." \
      io.openshift.tags="operator,shiftwise,kubeoptix,ai,golang" \
      maintainer="ShiftWise AI" \
      com.redhat.component="shiftwise-operator" \
      io.openshift.expose-services=""

COPY --from=builder /etc/pki/tls/certs/ca-bundle.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /opt/app-root/src/manager /usr/local/bin/manager

USER 65532:65532

WORKDIR /

ENTRYPOINT ["/usr/local/bin/manager"]
