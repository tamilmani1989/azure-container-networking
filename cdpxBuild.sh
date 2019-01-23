#!/bin/bash 
set -e

# Save current working directory
PWD=`pwd`
pushd $PWD

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get our Go compiler
GODIR="$DIR/go1-10"
if [ ! -d $GODIR ]; then
    mkdir -p $GODIR
    wget https://storage.googleapis.com/golang/go1.10.2.linux-amd64.tar.gz
    tar -C $GODIR -xvf go1.10.2.linux-amd64.tar.gz
    rm go1.10.2.linux-amd64.tar.gz
fi

# Set env variables
export GOOS=$2
export GOPATH=$DIR
export GOROOT="$GODIR/go"
export PATH=$GOROOT/bin:$PATH

case $1 in
    "build" )
        cd $DIR/src/dnc

        # Build go project and write the output to build.log
        go get ./...

        # Target OS specific parameters.
        if [ "$GOOS" = "windows" ];then
            # Windows.
            go build -o azure-dnc.exe >& build.log
        else
            # Linux.
            go build -o azure-dnc >& build.log
        fi
        ;;
    "package" )
        cd $DIR/Nuget/AzureDNCPackage
        mono nuget.exe install
        mono nuget.exe pack AzureDNCPackage.nuspec
        ;;
    "test" )
        cd $DIR/src/dnc
        export DNC_CONFIGURATION_PATH=$DIR/src/dnc/config_test.json
        ./azure-dnc &
        go test -timeout 1000s
        kill -SIGINT $(ps aux | grep './azure-dnc' | awk '{print $2}')
        ;;
esac

# Save the exit code from make
EX=$?

# Check exit code from Makefile and exit with it to ensure build fails
if [ "$EX" -ne "0" ]; then
    popd
    echo Failed to build DNC
fi

# Restore working directory
popd

# Exit with explicit 0 exit code to make sure build passes
exit $EX