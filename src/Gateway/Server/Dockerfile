FROM ubuntu:19.10

ARG GATEWAY_BUILD_DIR

COPY $GATEWAY_BUILD_DIR /app

WORKDIR /app

RUN chmod +x DncGateway; \
    apt-get update; \
    apt-get upgrade; \
	apt-get install -y ca-certificates; \
    apt-get install -y libssl-dev;

EXPOSE 5001

ENTRYPOINT ["/app/DncGateway"]

