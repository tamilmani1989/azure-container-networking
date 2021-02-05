# Build requestcontroller
# docker build -t nmagentmock:latest -f builds/nmagentmock/Dockerfile .
FROM golang:1.14-alpine AS builder

WORKDIR /usr/local/src/nmagentmock

# Copy the source, we don't have to be picky here since this is an intermediate container
COPY . .

# Fix the issue when building this Docker image from a Windows host, the filesystem for the vendor directory will be messed up
# We need the ApplicationInsights-Go module to be under vendor/github.com/microsoft
RUN if [[ -d vendor/github.com/Microsoft/ApplicationInsights-Go ]]; then \
	mkdir vendor/github.com/microsoft && \
	cp -r vendor/github.com/Microsoft/ApplicationInsights-Go \
	vendor/github.com/microsoft/ApplicationInsights-Go; fi

# Build
RUN cd test/nmagent/ && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -a -o /usr/local/bin/nmagentmock .

# Copy into final image
# For now, using alpine for ease of debugging, but this can be switched to scratch for smaller image size
FROM alpine:latest
COPY --from=builder /usr/local/bin/nmagentmock \
	/usr/local/bin/nmagentmock

ENTRYPOINT [ "/usr/local/bin/nmagentmock" ]
