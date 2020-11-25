FROM debian:10

ARG VERSION=9.0.3

# Ensure we get all the updates in case the debian upstream image isn't baked with them
RUN apt-get update && apt-get upgrade -y

# Build the Bond compiler
WORKDIR /usr/local/bond
RUN apt-get install -y \
    git \
    clang \
    cmake \
    zlib1g-dev \
    libboost-dev \
    libboost-thread-dev \
    curl
RUN git clone --recursive https://github.com/microsoft/bond.git /usr/local/bond && git checkout ${VERSION}
RUN curl -sSL https://get.haskellstack.org/ | sh
RUN mkdir build && \
    cd build && \
    cmake -DBOND_ENABLE_GRPC=FALSE .. && \
    make && \
    make install

# Set workdir back to root
WORKDIR /root