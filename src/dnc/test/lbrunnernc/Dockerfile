FROM alpine:3.8
ARG LBRUNNERNC_BUILD_DIR

COPY $LBRUNNERNC_BUILD_DIR/lbrunnernc lbrunnernc

RUN chmod +x lbrunnernc; \
    apk update; \
    apk add --no-cache tcpdump; \
    apk add --no-cache curl; \
    apk add --no-cache bash;

EXPOSE 9095

ENTRYPOINT ["/lbrunnernc"]

