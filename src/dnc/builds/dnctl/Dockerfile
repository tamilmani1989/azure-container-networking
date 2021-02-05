# Build runner
FROM golang:1.14-alpine AS builder

RUN apk --update add ca-certificates

WORKDIR /usr/local/src/dnc

COPY . .

# Fix vendoring issue when building on Windows host
RUN if [ -d ./vendor/github.com/Microsoft/ApplicationInsights-Go ]; then \
    mkdir -p ./vendor/github.com/microsoft/ && \
    mv ./vendor/github.com/Microsoft/ApplicationInsights-Go ./vendor/github.com/microsoft/; fi

# Build
RUN cd cmd/dnctl && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/dnctl .

# Copy into final image
FROM alpine:latest

WORKDIR /root

RUN echo '{"DncEP": "http://dnc-service:9000"}' > .dncclientconfig.json

COPY --from=builder /usr/local/bin/dnctl /usr/local/bin/dnctl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt


