FROM quay.io/operator-framework/opm:latest

COPY catalog/shiftwise-operator/catalog.yaml /configs/catalog.yaml

ENTRYPOINT ["opm", "serve", "/configs"]