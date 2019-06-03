FROM alpine:3.8

COPY swiftrunner swiftrunner

COPY container0.yaml container0.yaml

COPY container1.yaml container1.yaml

COPY cns.yaml cns.yaml

COPY dnc.yaml dnc.yaml

RUN chmod +x swiftrunner; \
    apk add --no-cache curl; \
    apk add --no-cache bash; \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl; \
    chmod +x ./kubectl; \
    mv ./kubectl /usr/local/bin/kubectl

ENTRYPOINT ["/swiftrunner"]

