#!/bin/sh
: '
Usage: sh collect-cnilogs.sh "<blob-sas-url>"
The script collects logs and state files and compress it as a tar gz file. It then uploads the tar.gz file to specific blob url.
'

set -e
mkdir -p /tmp/azurecni-diagnose/
cp -rf /var/log/azure-* /tmp/azurecni-diagnose
cp -rf /var/run/azure-* /tmp/azurecni-diagnose
cp /var/log/cilium-cni* /tmp/azurecni-diagnose
cp -rf /var/log/pods/kube-system_cilium-* /tmp/azurecni-diagnose
cd /tmp
tar -cvzf azurecni-diagnose.tar.gz azurecni-diagnose/
rm -rf azurecni-diagnose
mkdir -p azcopy
wget https://aka.ms/downloadazcopy-v10-linux -O azcopy.tar
tar -xvf azcopy.tar -C azcopy/
mv azcopy/azcopy*/azcopy azcopy/azcopy
./azcopy/azcopy copy azurecni-diagnose.tar.gz $1
rm -rf azcopy*
rm -rf azurecni-*