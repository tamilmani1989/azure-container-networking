FROM alpine:3.8

COPY dnc dnc

COPY azure-vnet-telemetry azure-vnet-telemetry

COPY config.json config.json

RUN chmod +x dnc; \
    apk add --no-cache bash; \
    apk add --no-cache curl

EXPOSE 9000

ENTRYPOINT ["/dnc"]

