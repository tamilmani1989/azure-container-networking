FROM alpine:3.8
ARG DNC_BUILD_DIR

COPY $DNC_BUILD_DIR/dnc dnc

COPY config.json config.json

RUN chmod +x dnc; \
    apk update; \
    apk add --no-cache bash; \
    apk add --no-cache curl; \
    apk add ca-certificates; \
    rm -rf /var/cache/apk/*; \
    sed -i 's/#LeaderElectionTest#/""/g' config.json; \
    sed -i 's/#LeaderElectionBlob#/""/g' config.json; \
    sed -i 's/"ReverseProxyEnabled" : false/"ReverseProxyEnabled" : true/g' config.json

EXPOSE 9000

ENTRYPOINT ["/dnc"]

