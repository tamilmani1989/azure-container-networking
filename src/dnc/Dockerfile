FROM ubuntu:19.10
ARG DNC_BUILD_DIR="."

COPY $DNC_BUILD_DIR/dnc dnc

COPY config.json config.json

RUN chmod +x dnc; \
    apt-get update; \
    apt-get upgrade; \
    apt-get install -y ca-certificates; \
    sed -i 's/#LeaderElectionTest#/""/g' config.json; \
    sed -i 's/#LeaderElectionBlob#/""/g' config.json; \
    sed -i 's/"ReverseProxyEnabled" : false/"ReverseProxyEnabled" : true/g' config.json

EXPOSE 9000

ENTRYPOINT ["/dnc"]

