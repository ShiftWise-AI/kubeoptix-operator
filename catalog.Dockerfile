FROM registry.access.redhat.com/ubi9/go-toolset:1.25 AS builder

ENV CGO_ENABLED=0 \
	GOTOOLCHAIN=go1.26.9

WORKDIR /opt/app-root/src

RUN go mod download github.com/operator-framework/operator-registry@v1.74.0 \
	&& cp -R "$(go env GOPATH)/pkg/mod/github.com/operator-framework/operator-registry@v1.74.0" /opt/app-root/src/opm-build \
	&& chmod -R u+w /opt/app-root/src/opm-build \
	&& cd /opt/app-root/src/opm-build \
	&& go get golang.org/x/text@v0.41.0 golang.org/x/crypto@v0.55.0 \
		golang.org/x/net@v0.58.0 google.golang.org/grpc@v1.83.2 \
		github.com/go-git/go-git/v5@v5.19.2 \
	&& go build -tags containers_image_openpgp -trimpath -ldflags '-s -w' -o /opt/app-root/src/opm \
		./cmd/opm \
	&& GOBIN=/opt/app-root/src go install github.com/grpc-ecosystem/grpc-health-probe@v0.4.59 \
	&& mv /opt/app-root/src/grpc-health-probe /opt/app-root/src/grpc_health_probe

FROM quay.io/operator-framework/opm@sha256:6bd04d9a3d35bc083de18c5bf6cfca73056d255053507abecaacda2a2d7aeda2

COPY --from=builder /opt/app-root/src/opm /usr/bin/opm
COPY --from=builder /opt/app-root/src/grpc_health_probe /usr/bin/grpc_health_probe
COPY catalog /configs

USER 65532:65532

ENTRYPOINT ["opm"]
CMD ["serve", "/configs"]
