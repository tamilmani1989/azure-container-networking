#!/bin/bash

set -e

echo '-------- Common Restore ----------------'
# Save current working directory
PWD=`pwd`
pushd "${PWD}" 1>> /dev/null

# Figure out location of this script. Don't assume absolute paths because paths can change between pipeline host and dev box.
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
cd "${DIR}/.."

# Install build-dependencies
apt update
apt install -y --no-install-recommends clang cmake zlib1g-dev libboost-dev libboost-thread-dev gcc gdb build-essential libssl-dev

# Install Haskell Tool Stack
echo '-------- Install Haskell Tool Stack ----------------'

curl -sSL https://get.haskellstack.org/ | sh -s - -f

echo '-------- Download Bond ----------------'

# turn the detached message off
git config --global advice.detachedHead false

git clone --recursive --branch 8.0.1 https://github.com/Microsoft/bond.git

echo '-------- Build Bond ----------------'
cd bond/
mkdir build
cd build/
cmake -DBOND_ENABLE_GRPC=FALSE ..
make -j8
make install

EX=$?
if [ "$EX" -ne "0" ]; then 
    echo "Bond failed." 1>&2
    exit $EX
fi

# Restore working directory
popd 1>> /dev/null

exit $?