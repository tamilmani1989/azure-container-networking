FROM alpine:3.8
ARG RUNNER_BUILD_DIR

COPY $RUNNER_BUILD_DIR/runner runner

COPY test/runner/container1.yaml container1.yaml

COPY test/runner/container2.yaml container2.yaml

COPY test/runner/cns.yaml cns.yaml

COPY test/runner/dnc.yaml dnc.yaml

RUN chmod +x runner; \
    apk update; \
    apk add --no-cache curl; \
    apk add --no-cache bash; \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl; \
    chmod +x ./kubectl; \
    mv ./kubectl /usr/local/bin/kubectl

ENTRYPOINT ["/runner"]

