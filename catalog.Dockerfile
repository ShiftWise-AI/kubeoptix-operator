FROM quay.io/operator-framework/opm:latest

COPY catalog /configs

ENTRYPOINT ["opm"]
CMD ["serve", "/configs"]
