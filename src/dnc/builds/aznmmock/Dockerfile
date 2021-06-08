# Steps to build mock aznm image
#
# cd Networking-Aquarius/src
# docker build -t mock-aznm:version -f dnc/builds/aznmmock/Dockerfile --no-cache .
#

# Build the Mock AZNM binary
# golang:1.14 is based on debian 10
FROM golang:1.14 AS mock-builder
WORKDIR /usr/local/src/Networking-Aquarius/src
COPY ./dnc ./dnc
#COPY ./dnc/microsoft/azure/aznm/contracts ./dnc/microsoft/azure/aznm/contracts

ENV GOPATH=/usr/local/src/Networking-Aquarius
RUN cd ./dnc/cmd/aznm && \
    GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -o /usr/local/bin/mock-aznm

FROM gcr.io/distroless/cc-debian10:latest
COPY --from=mock-builder /usr/local/bin/mock-aznm /usr/local/bin/mock-aznm

ENTRYPOINT [ "/usr/local/bin/mock-aznm" ]