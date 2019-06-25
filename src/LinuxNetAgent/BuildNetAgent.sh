#!/bin/bash
set -e
scriptdir=$(dirname "$0")
export GOPATH=$scriptdir/../..
rm -rf  $scriptdir/../github.com
rm -rf  $scriptdir/../golang.org
go get "golang.org/x/sys/unix"
go get "github.com/Azure/azure-container-networking/cnm"
cd $scriptdir/main
go build -o servicetunnel
chmod +x servicetunnel
chmod +x $scriptdir/scripts/*.sh

