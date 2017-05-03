#!/bin/bash
srcroot=`pwd`
set -e
cd src/NetworkClientLibrary/Linux
rm -rf *.o *.so *so.* a.out
cd $srcroot
rm -rf out
rm -rf src/azure-overlay/output
rm -rf src/test/Linux/*.o
rm -rf src/test/Linux/NetworkClientTest
