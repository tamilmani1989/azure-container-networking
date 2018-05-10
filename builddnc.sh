#!/bin/bash 
set -e
srcroot=`pwd`
export GOPATH=`pwd`
cd $srcroot/src
GOOS=$1 GOARCH=$2 make all-binaries


