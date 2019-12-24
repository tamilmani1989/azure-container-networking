#!/bin/bash
set -e
GODIR="$DIR/go1-12"
if [ ! -d $GODIR ]; then
    mkdir -p $GODIR
    wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
    tar -C $GODIR -xvf go1.13.3.linux-amd64.tar.gz
    rm go1.13.3.linux-amd64.tar.gz
fi

# Set env variables
export GOROOT="$GODIR/go"
export PATH=$GOROOT/bin:$PATH
scriptdir="$( cd "$(dirname "$0")" ; pwd -P )"
export GOPATH=$scriptdir/../..

go get "golang.org/x/sys/unix"
go get "github.com/Azure/azure-container-networking/cns"
cd $scriptdir/servicetunnel
go test -run "TestCloud*"
EX=$?
if [ $EX -ne 0 ]; then
  exit $EX
fi

cd $scriptdir/configuration
go test
EX=$?
exit $EX