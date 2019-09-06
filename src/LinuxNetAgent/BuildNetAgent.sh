#!/bin/bash
set -e
GODIR="$DIR/go1-12"
if [ ! -d $GODIR ]; then
    mkdir -p $GODIR
    wget https://storage.googleapis.com/golang/go1.12.6.linux-amd64.tar.gz
    tar -C $GODIR -xvf go1.12.6.linux-amd64.tar.gz
    rm go1.12.6.linux-amd64.tar.gz
fi

# Set env variables
export GOROOT="$GODIR/go"
export PATH=$GOROOT/bin:$PATH
scriptdir="$( cd "$(dirname "$0")" ; pwd -P )"
export GOPATH=$scriptdir/../..

rm -rf  $scriptdir/../github.com
rm -rf  $scriptdir/../golang.org
go version
export GO111MODULE=off
go get "golang.org/x/sys/unix"
go get "github.com/Azure/azure-container-networking/cnm"
cd $scriptdir/main
go build -o servicetunnel
chmod +x servicetunnel
chmod +x $scriptdir/scripts/*.sh

