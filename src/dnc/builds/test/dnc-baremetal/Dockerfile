# This Dockerfile needs access to the entire Networking-Aquarius repo, so when building, ensure to pass the
# repo root as the build context. For example:
#
#   cd Networking-Aquarius/
#   docker build -t dnc:version -f src/dnc/builds/test/dnc-baremetal/Dockerfile --no-cache .
#
# If building locally, you can use the cache to speed up builds:
#
#   cd Networking-Aquiarius/
#   docker build -t dnc:version -f src/dnc/builds/test/dnc-baremetal/Dockerfile .
#
# If building locally and you want to skip building bondserializer (since it takes too long to build), then use this.
# It will use the checked-in version of bond serializer library rather than building it fresh.
#   docker build -t dnc:version -f src/dnc/builds/test/dnc-baremetal/Dockerfile --build-arg GENERATE_BOND_FILE=false .

# Get a trusted CA cert bundle and go toolchain
FROM alpine:latest as certs
RUN apk --update add ca-certificates

# Build the libBondDataSerializer.so file conditionally. Default behavior is to build this file.
# However, if caller passes in GENERATE_BOND_FILE=false as build argument in docker cli, this file
# won't be built and checked-in version of libBondDataSerializer.so be used from repo.
FROM mcr.microsoft.com/containernetworking/gbc:9.0.3 AS bond-builder
WORKDIR /usr/local/src/Networking-Aquarius
ARG GENERATE_BOND_FILE=true
COPY ./src/BondDataSerializer ./src/BondDataSerializer
COPY ./src/dnc/pubsubproxy/libs ./src/dnc/pubsubproxy/libs
RUN if [ "$GENERATE_BOND_FILE" = "true" ]; then \
      cd ./src/BondDataSerializer && \
      rm -rf build && \
      mkdir build && \
      cd build && \
      cmake .. && \
      make; \
	else \
	  cp -r ./src/dnc/pubsubproxy/libs ./src/BondDataSerializer/build ; \
	fi

# Build the DNC binary
# golang:1.16 is based on debian 10
FROM golang:1.16 AS dnc-builder
WORKDIR /usr/local/src/Networking-Aquarius
ENV LD_LIBRARY_PATH /usr/local/src/Networking-Aquarius/src/dnc/pubsubproxy/libs
ENV CGO_LDFLAGS "-L ${LD_LIBRARY_PATH} -lMadariNativeClient -lBondDataSerializer"
COPY ./src/dnc ./src/dnc
# Copy the new libBondDataSerializer from the bond-builder step over top of the existing file in the repo
COPY --from=bond-builder /usr/local/src/Networking-Aquarius/src/BondDataSerializer/build/libBondDataSerializer.so ${LD_LIBRARY_PATH}
ARG VERSION=dirty
ARG aiMetadata="4a542967-4721-4c64-a2a7-eb9428c634b4"
# Fix vendoring issue when building on Windows host
RUN if [ -d ./src/dnc/vendor/github.com/Microsoft/ApplicationInsights-Go ]; then \
    mkdir -p ./src/dnc/vendor/github.com/microsoft/ && \
    mv ./src/dnc/vendor/github.com/Microsoft/ApplicationInsights-Go ./src/dnc/vendor/github.com/microsoft/; fi
RUN cd ./src/dnc && \
    GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -tags enablePubsubProxy -o /usr/local/bin/azure-dnc \
    -ldflags "-X dnc/utils.Version=$VERSION -X dnc/aiwrapper.aiMetadata=$aiMetadata -s -w"

# Copy DNC binary, Bond .so files, and trusted cert bundle into final image
# Using distroless/cc-debian10 because it is minimal but provides glibc and libstdc++
# We need -debian10 to get the most up-to-date C and C++ libs
FROM debian:10
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=dnc-builder /usr/local/bin/azure-dnc /usr/local/bin/azure-dnc
COPY --from=dnc-builder /usr/local/src/Networking-Aquarius/src/dnc/pubsubproxy/libs/ /usr/local/lib
COPY --from=dnc-builder /usr/local/src/Networking-Aquarius/src/dnc/acnkv-CertDncPubsub-20201001.pem /usr/local/bin/
COPY --from=dnc-builder /usr/local/src/Networking-Aquarius/src/dnc/config_test_baremetal_local.json /usr/local/bin/

RUN apt update && apt install -y gcc libssl-dev

ENV DNC_CONFIGURATION_PATH /usr/local/bin/config_test_baremetal_local.json

ENV LD_LIBRARY_PATH /usr/local/lib

ENTRYPOINT [ "/usr/local/bin/azure-dnc" ]