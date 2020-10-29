# Build runner
FROM golang:1.14-alpine AS builder

RUN apk --update add ca-certificates

WORKDIR /usr/local/src/dnc

COPY . .

# Build
RUN cd cmd/dnctl && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/dnctl .

# Copy into final image
FROM alpine:latest

WORKDIR /root

RUN echo '{"DncEP": "http://dnc-service:9000"}' > .dncclientconfig.json

COPY --from=builder /usr/local/bin/dnctl /usr/local/bin/dnctl
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt


