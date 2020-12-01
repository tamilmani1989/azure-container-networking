#!/bin/bash

set -e

# as of 1.14, the go command defaults to using a vendor folder if present in the main module

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

cd "$SCRIPTDIR"/main

VERSION="1.7"
go build -o servicetunnel -ldflags "-X main.version=$VERSION -s -w"
chmod +x servicetunnel


