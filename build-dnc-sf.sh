#!/bin/bash 
set -e

# Save current working directory
PWD=`pwd`
pushd $PWD

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
export DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $DIR

Operation=$1
BuildConfig=$2

if [ "$Operation"="Build" ]; then
    echo '---------- Restore DNC SF projects ----------'
    dotnet restore "src/Dnc-SF/DNCService.Interfaces/DNCService.Interfaces.csproj"
    dotnet restore "src/Dnc-SF/DNCService/DNCService.csproj"

    # Save the exit code from make
    EX=$?

    # Check exit code from Makefile and exit with it to ensure build fails
    if [ "$EX" -ne "0" ]; then
        popd
        echo Failed to restore DNC SF
    fi

    echo '---------- Build DNC SF projects ----------'
    dotnet build "src/Dnc-SF/DNCService.Interfaces/DNCService.Interfaces.csproj" --configuration $BuildConfig
    dotnet build "src/Dnc-SF/DNCService/DNCService.csproj" --configuration $BuildConfig

    # Save the exit code from make
    EX=$?

    # Check exit code from Makefile and exit with it to ensure build fails
    if [ "$EX" -ne "0" ]; then
        popd
        echo Failed to build DNC SF
    fi

    echo '---------- Prepare DNC SF Application ----------'
    nuget restore "src/Dnc-SF/DncSFProj/packages.config" -PackagesDirectory "src/Dnc-SF/DncSFProj/packages"
    dotnet msbuild "src/Dnc-SF/DncSFProj/DncApplication.sfproj" -p:Configuration=$BuildConfig /t:Package

    # Save the exit code from make
    EX=$?

    # Check exit code from Makefile and exit with it to ensure build fails
    if [ "$EX" -ne "0" ]; then
        popd
        echo Failed to prepare DNC SF Application
    fi
fi

# Restore working directory
popd

# Exit with explicit 0 exit code to make sure build passes
exit $EX