#!/bin/bash 
set -e

# Save current working directory
PWD=`pwd`
pushd $PWD

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get our Go compiler
GODIR="$DIR/go1-12"
if [ ! -d $GODIR ]; then
    mkdir -p $GODIR
    wget https://storage.googleapis.com/golang/go1.12.6.linux-amd64.tar.gz
    tar -C $GODIR -xvf go1.12.6.linux-amd64.tar.gz
    rm go1.12.6.linux-amd64.tar.gz
fi

# Set env variables
export GOOS=$2
export GOPATH=$DIR
export GOROOT="$GODIR/go"
export PATH=$GOROOT/bin:$PATH
export VERSION=$CDP_MAJOR_NUMBER_ONLY.$CDP_MINOR_NUMBER_ONLY.$CDP_VERSION_TAG_ONLY

case $1 in
    "build" )
        cd $DIR/src/dnc

        # Build go project and write the output to build.log
        go get ./...

        # Target OS specific parameters.
        if [ "$GOOS" = "windows" ];then
            # Windows.
            go build -o azure-dnc.exe -ldflags "-X dnc/utils.Version=$VERSION -s -w" >& build.log
        else
            # Linux.
            go build -o azure-dnc -ldflags "-X dnc/utils.Version=$VERSION -s -w" >& build.log
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
        go test -timeout 2000s -coverprofile coverage-dnc.out
        ;;
    "get" )
        mkdir $DIR/src/github && cd $_
        latestVersion=$(curl -L -s -H 'Accept: application/json' https://github.com/Azure/azure-container-networking/releases/latest | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
        # Get latest version of CNI
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-multitenancy-windows-amd64-$latestVersion.zip
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-multitenancy-linux-amd64-$latestVersion.tgz
        unzip azure-vnet-cni-multitenancy-windows-amd64-$latestVersion.zip
        mv 10-azure.conflist 10-azure-win.conflist
        tar -xvzf azure-vnet-cni-multitenancy-linux-amd64-$latestVersion.tgz
        # Get latest version of CNS
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-cns-windows-amd64-$latestVersion.zip
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-cns-linux-amd64-$latestVersion.tgz
        unzip azure-cns-windows-amd64-$latestVersion.zip
        tar -xvzf azure-cns-linux-amd64-$latestVersion.tgz
        rm *.zip
        rm *.tgz
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