# Build nmagentmockv2
# docker build -t nmagentmockv2:latest -f builds/nmagentmockv2/Dockerfile .
FROM golang:1.16-alpine AS builder

WORKDIR /usr/local/src/nmagentmock

# Copy the source, we don't have to be picky here since this is an intermediate container
COPY . .

# Build
RUN cd cmd/nmagentmock/ && \
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/nmagentmock .

# Copy into final image
# For now, using alpine for ease of debugging, but this can be switched to scratch for smaller image size
FROM alpine:latest
COPY --from=builder /usr/local/bin/nmagentmock \
	/usr/local/bin/nmagentmock

ENTRYPOINT [ "/usr/local/bin/nmagentmock" ]
