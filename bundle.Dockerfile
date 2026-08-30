FROM scratch

LABEL operators.operatorframework.io.bundle.manifests.v1=manifests/
LABEL operators.operatorframework.io.bundle.metadata.v1=metadata/
LABEL operators.operatorframework.io.bundle.package.v1=shiftwise-operator
LABEL operators.operatorframework.io.bundle.channels.v1=stable
LABEL operators.operatorframework.io.bundle.channel.default.v1=stable

COPY bundle/manifests/shiftwise.ai_shiftwises.yaml /manifests/
COPY bundle/manifests/shiftwise-operator.v0.1.1.clusterserviceversion.yaml /manifests/
COPY bundle/metadata /metadata/
