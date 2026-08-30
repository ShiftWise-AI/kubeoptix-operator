FROM docker.io/library/lua:5.4-bookworm

USER 0

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates curl lua-dkjson \
		&& curl -fsSL --retry 3 -o /usr/local/bin/kubectl \
			https://dl.k8s.io/release/v1.32.1/bin/linux/amd64/kubectl \
		&& chmod 0755 /usr/local/bin/kubectl \
		&& rm -rf /var/lib/apt/lists/* \
		&& mkdir -p /opt/shiftwise-operator \
		&& chown -R 1001:0 /opt/shiftwise-operator

COPY lua/ /opt/shiftwise-operator/

USER 1001

ENTRYPOINT ["lua", "/opt/shiftwise-operator/shiftwise_operator.lua"]
