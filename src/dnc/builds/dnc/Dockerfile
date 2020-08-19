# This Dockerfile attempts to put all of the network operations, dependency gathering, and
# other tasks that are not compiling DNC toward the top of this file to make use of layer caching.
# This means that locally, builds should be quick. However, to ensure the images in ACR are built with
# the least number of vulnerabilities, the pipeline will build without the cache. If you plan on building
# locally and then pushing to ACR, you should also build without cache, like so:
#
#   cd Networking-Aquarius/
#   docker build -t dnc:version -f src/dnc/builds/dnc/Dockerfile --no-cache .

# Get a trusted CA cert bundle
FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM debian:10 AS builder

# Build the Bond compiler
WORKDIR /usr/local/bond
RUN apt-get update && apt-get install -y \
    git \
    clang \
    cmake \
    zlib1g-dev \
    libboost-dev \
    libboost-thread-dev \
    curl
RUN git clone --recursive https://github.com/microsoft/bond.git /usr/local/bond
RUN curl -sSL https://get.haskellstack.org/ | sh
RUN mkdir build && \
    cd build && \
    cmake -DBOND_ENABLE_GRPC=FALSE .. && \
    make && \
    make install

# Install Go toolchain
WORKDIR /root
RUN curl -sSLO https://golang.org/dl/go1.14.6.linux-amd64.tar.gz && \
    tar -C /usr/local -xzf go1.14.6.linux-amd64.tar.gz
WORKDIR /usr/local/src/Networking-Aquarius
ENV LD_LIBRARY_PATH /usr/local/src/Networking-Aquarius/src/dnc/pubsubproxy/libs
ENV CGO_LDFLAGS "-L ${LD_LIBRARY_PATH} -lMadariNativeClient -lBondDataSerializer"

# Copy the BondDataSerializer source
COPY ./src/BondDataSerializer ./src/BondDataSerializer

# Build BondDataSerializer
RUN cd ./src/BondDataSerializer && \
    rm -rf build && \
    mkdir build && \
    cd build && \
    cmake .. && \
    make && \
    mkdir -p ${LD_LIBRARY_PATH} && \
    cp libBondDataSerializer.so ${LD_LIBRARY_PATH}

# Copy the DNC source
# This happens here so that everything above should be cached when building locally,
# since the DNC source code is likely to be the thing that changes the most.
COPY ./src/dnc ./src/dnc

# Build DNC
ARG VERSION=dirty
ARG aiMetadata="4a542967-4721-4c64-a2a7-eb9428c634b4"
RUN cd ./src/dnc && \
    GOOS=linux GOARCH=amd64 /usr/local/go/bin/go build -tags enablePubsubProxy -o /usr/local/bin/azure-dnc \
    -ldflags "-X dnc/utils.Version=$VERSION -X dnc/aiwrapper.aiMetadata=$aiMetadata -s -w"

# Copy DNC binary, Bond .so files, and trusted cert bundle into final image
# Using distroless/cc-debian10 because it is minimal but provides glibc and libstdc++
# We need -debian10 to get the most up-to-date C and C++ libs
FROM gcr.io/distroless/cc-debian10
COPY --from=builder /usr/local/bin/azure-dnc /usr/local/bin/azure-dnc
COPY --from=builder /usr/local/src/Networking-Aquarius/src/dnc/pubsubproxy/libs/ /usr/local/lib
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENV LD_LIBRARY_PATH /usr/local/lib

ENTRYPOINT [ "/usr/local/bin/azure-dnc" ]