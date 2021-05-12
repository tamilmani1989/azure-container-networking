# Build runner
FROM golang:1.14-alpine AS builder

WORKDIR /usr/local/src/dnc

COPY . .

# Fix the issue when building this Docker image from a Windows host, the filesystem for the vendor directory will be messed up
# We need the ApplicationInsights-Go module to be under vendor/github.com/microsoft
RUN if [[ -d vendor/github.com/Microsoft/ApplicationInsights-Go ]]; then \
	mkdir vendor/github.com/microsoft && \
	cp -r vendor/github.com/Microsoft/ApplicationInsights-Go \
	vendor/github.com/microsoft/ApplicationInsights-Go; fi

# Build
RUN cd test/runner && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/runner .

# Copy into final image
FROM alpine:latest

WORKDIR /etc/runner/

COPY --from=builder /usr/local/bin/runner /usr/local/bin/runner
COPY --from=builder /usr/local/src/dnc/test/runner/*.yaml ./

RUN apk update; \
    apk add --no-cache curl; \
    apk add --no-cache bash; \
    curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl; \
    chmod +x ./kubectl; \
    mv ./kubectl /usr/local/bin/kubectl

ENTRYPOINT ["/usr/local/bin/runner"]

