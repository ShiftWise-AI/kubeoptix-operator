FROM quay.io/fedora/fedora:41

USER 0

RUN dnf install -y lua luarocks curl ca-certificates \
	&& luarocks --lua-version=5.4 install dkjson \
		&& curl -fsSL --retry 3 -o /usr/local/bin/kubectl \
			https://dl.k8s.io/release/v1.32.1/bin/linux/amd64/kubectl \
		&& chmod 0755 /usr/local/bin/kubectl \
		&& dnf clean all \
		&& mkdir -p /opt/shiftwise-operator \
		&& chown -R 1001:0 /opt/shiftwise-operator

COPY lua/ /opt/shiftwise-operator/

USER 1001

ENTRYPOINT ["lua", "/opt/shiftwise-operator/shiftwise_operator.lua"]
