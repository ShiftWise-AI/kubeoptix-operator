FROM quay.io/operator-framework/helm-operator:v1.38.0

USER root
COPY watches.yaml ${HOME}/watches.yaml
COPY helm-charts/ ${HOME}/helm-charts/
RUN chown -R 1001:0 ${HOME}/watches.yaml ${HOME}/helm-charts
USER 1001
