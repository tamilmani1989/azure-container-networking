# Build csmock
# docker build -t cnsmock:latest -f builds/cnsmock/Dockerfile ./test/cnsmock
FROM golang:1.14-alpine AS builder

RUN apk add git

COPY . /go/src/cnsmock
WORKDIR /go/src/cnsmock

# Build
RUN go get ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o /usr/local/bin/cnsmock .

# Copy into final image
# For now, using alpine for ease of debugging, but this can be switched to scratch for smaller image size
FROM scratch
COPY --from=builder /usr/local/bin/cnsmock \
	/usr/local/bin/cnsmock

ENTRYPOINT [ "/usr/local/bin/cnsmock" ]
