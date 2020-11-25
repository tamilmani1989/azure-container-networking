# Build requestcontroller
FROM golang:1.14-alpine AS builder

# Get deps
RUN apk --update add ca-certificates curl unzip

# Get kubelogin binary
WORKDIR /usr/local/kubelogin
RUN curl -sSLO https://github.com/Azure/kubelogin/releases/download/v0.0.6/kubelogin-linux-amd64.zip && \
	unzip kubelogin-linux-amd64.zip

WORKDIR /usr/local/src/dnc

# Copy the source, we don't have to be picky here since this is an intermediate container
COPY . .

# Fix the issue when building this Docker image from a Windows host, the filesystem for the vendor directory will be messed up
# We need the ApplicationInsights-Go module to be under vendor/github.com/microsoft
RUN if [[ -d vendor/github.com/Microsoft/ApplicationInsights-Go ]]; then \
	mkdir vendor/github.com/microsoft && \
	cp -r vendor/github.com/Microsoft/ApplicationInsights-Go \
	vendor/github.com/microsoft/ApplicationInsights-Go; fi

# Build
RUN cd cmd/requestcontroller && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -a -o /usr/local/bin/requestcontroller .

# Copy into final image
# For now, using alpine for ease of debugging, but this can be switched to scratch for smaller image size
FROM alpine:latest
COPY --from=builder /usr/local/bin/requestcontroller \
	/usr/local/bin/requestcontroller
COPY --from=builder /usr/local/kubelogin/bin/linux_amd64/kubelogin /usr/local/bin/kubelogin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

ENTRYPOINT [ "/usr/local/bin/requestcontroller" ]
