# Steps to build and run mock macallocator image
#
# cd Networking-Aquarius
# docker build -t macmock:latest -f .\src\dnc\builds\macallocatormock\Dockerfile --no-cache .
# docker run -p 5012:5012 mockmacallocator:tag -address 0.0.0.0:5012

# Build the Mock MacAllocator binary
# golang:1.14 is based on debian 10
FROM golang:1.14 AS mock-builder
WORKDIR /usr/local/src/Networking-Aquarius/src
COPY . .

ENV GOPATH=/usr/local/src/Networking-Aquarius
RUN cd ./src/dnc/cmd/macallocatormock && \
    GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o /usr/local/bin/mock-macallocator

FROM debian:10
COPY --from=mock-builder /usr/local/bin/mock-macallocator /usr/local/bin/mock-macallocator

ENTRYPOINT [ "/usr/local/bin/mock-macallocator" ]