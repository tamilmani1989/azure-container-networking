FROM alpine:3.8

COPY netPerfRunner netPerfRunner

COPY iperfc.yaml iperfc.yaml

COPY iperfs.yaml iperfs.yaml

RUN chmod +x netPerfRunner; \
    apk add --no-cache curl; \
    apk add --no-cache bash; \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl; \
    chmod +x ./kubectl; \
    mv ./kubectl /usr/local/bin/kubectl

ENTRYPOINT ["/netPerfRunner"]

