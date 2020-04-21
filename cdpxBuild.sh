#!/bin/bash
set -e

# Save current working directory
PWD=`pwd`
pushd $PWD

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Get our Go compiler
GODIR="$DIR/go1-14"
if [ ! -d $GODIR ]; then
    mkdir -p $GODIR
    wget https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz
    tar -C $GODIR -xvf go1.14.2.linux-amd64.tar.gz
    rm go1.14.2.linux-amd64.tar.gz
fi

# Set env variables
export GOOS=$2
export GOPATH=$DIR
export GOROOT="$GODIR/go"
export PATH=$GOROOT/bin:$PATH
export VERSION=$CDP_MAJOR_NUMBER_ONLY.$CDP_MINOR_NUMBER_ONLY.$CDP_VERSION_TAG_ONLY
export aiMetadata="4a542967-4721-4c64-a2a7-eb9428c634b4"
case $1 in
    "build" )
        cd $DIR/src/dnc

        # Build go project and write the output to build.log
        # Target OS specific parameters.
        if [ "$GOOS" = "windows" ];then
            # Windows.
            go build -o azure-dnc.exe -ldflags "-X dnc/utils.Version=$VERSION -X dnc/aiwrapper.aiMetadata=$aiMetadata -s -w" >& build.log
        else
            # Linux.
            go build -o azure-dnc -ldflags "-X dnc/utils.Version=$VERSION -X dnc/aiwrapper.aiMetadata=$aiMetadata -s -w" >& build.log
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
        go test -run "[^*TestCreateMultipleNetworks*]" -timeout 2000s -coverprofile coverage-dnc.out
        zip azure-dnc-test-logs.zip /var/log/azure-dnc*
        ;;
    "loadtest" )
        cd $DIR/src/dnc
        export DNC_CONFIGURATION_PATH=$DIR/src/dnc/config_test.json
        go test -run "^(TestCreateMultipleNetworks)$" -timeout 2000s -coverprofile coverage-dnc.out
        zip azure-dnc-load-test-logs.zip /var/log/azure-dnc*
        ;;
    "get" )
        mkdir $DIR/src/github && cd $_
        latestVersion=$(curl -L -s -H 'Accept: application/json' https://github.com/Azure/azure-container-networking/releases/latest | sed -e 's/.*"tag_name":"\([^"]*\)".*/\1/')
        # Get latest version of multitenant CNI, only multitenant is pushed as a Nuget
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-multitenancy-windows-amd64-$latestVersion.zip
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-multitenancy-linux-amd64-$latestVersion.tgz
        unzip azure-vnet-cni-multitenancy-windows-amd64-$latestVersion.zip
        mv 10-azure.conflist 10-azure-windows-multitenancy.conflist
        tar -xvzf azure-vnet-cni-multitenancy-linux-amd64-$latestVersion.tgz
        mv 10-azure.conflist 10-azure-linux-multitenancy.conflist

        # Get latest version of singletenant CNI
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-windows-amd64-$latestVersion.zip
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-vnet-cni-linux-amd64-$latestVersion.tgz
        unzip -o azure-vnet-cni-windows-amd64-$latestVersion.zip
        mv 10-azure.conflist 10-azure-windows-singletenancy.conflist
        tar -xvzf azure-vnet-cni-linux-amd64-$latestVersion.tgz
        mv 10-azure.conflist 10-azure-linux-singletenancy.conflist

        # Get latest version of CNS
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-cns-windows-amd64-$latestVersion.zip
        curl -LO https://github.com/Azure/azure-container-networking/releases/download/$latestVersion/azure-cns-linux-amd64-$latestVersion.tgz
        unzip -o azure-cns-windows-amd64-$latestVersion.zip
        tar -xvzf azure-cns-linux-amd64-$latestVersion.tgz
        rm *.zip
        rm *.tgz
        ;;
    "configupdate" )
        cd $DIR/src/dnc
        sed 's/#LeaderElectionTest#/""/g' config.json | sed 's/#LeaderElectionBlob#/""/g' > sf_dnc_config.json
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