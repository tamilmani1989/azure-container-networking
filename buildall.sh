#!/bin/bash 
set -e
srcroot=`pwd`
export GOPATH=`pwd`
go get "golang.org/x/sys/unix"
cd src/azure-overlay
make all-binaries
cd $srcroot
export LD_LIBRARY_PATH=$LD_LIBRARY_PATH:./src/NetworkClientLibrary/Linux
cd src/NetworkClientLibrary/Linux
make && make client
cd $srcroot/src/test/Linux
make


