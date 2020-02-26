FROM alpine:3.8

RUN apk update; \
    apk upgrade; \
    apk add --no-cache bash; \
    apk add --no-cache curl; \
    apk add --no-cache bind-tools; \
    apk add ca-certificates; \
    rm -rf /var/cache/apk/*

ENTRYPOINT ["tail", "-f", "/dev/null"]
