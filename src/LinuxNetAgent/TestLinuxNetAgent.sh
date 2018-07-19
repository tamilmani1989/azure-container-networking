#!/bin/bash
set -e
scriptdir=$(dirname "$0")
export GOPATH=$scriptdir/../..
go get "golang.org/x/sys/unix"
go get "github.com/Azure/azure-container-networking/cns"
cd $scriptdir/servicetunnel
go test -run "TestCloud*"
EX=$?
if [ $EX -ne 0 ]; then
  exit $EX
fi

cd $scriptdir/ExtensionHandler
go test
EX=$?
exit $EX