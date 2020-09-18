#!/bin/bash

set -e
GODIR="$DIR/go1.14.9"

rm -rf "$GODIR"
mkdir -p "$GODIR"

wget https://golang.org/dl/go1.14.9.linux-amd64.tar.gz
tar -C "$GODIR" -xf go1.14.9.linux-amd64.tar.gz
rm go1.14.9.linux-amd64.tar.gz

SCRIPTDIR="$( cd "$(dirname "$0")" ; pwd -P )"

export PATH="$GODIR/go"/bin:$PATH

go version
go env

cd "$SCRIPTDIR"/servicetunnel

go test -run "TestCloud*"
EX=$?
if [ $EX -ne 0 ]; then
  exit $EX
fi

cd "$SCRIPTDIR"/ExtensionHandler
go test
EX=$?
exit $EX