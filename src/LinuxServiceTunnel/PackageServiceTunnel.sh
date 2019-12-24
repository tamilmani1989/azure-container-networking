#!/bin/bash
set -e
version="1.7"
scriptdir=$(dirname "$0")
outdir=$scriptdir/out/ServiceTunnelLinux
zippath="$scriptdir/out/ServiceTunnelLinux_$version.zip"
mkdir -p $outdir
cp $scriptdir/main/servicetunnel $outdir
cp $scriptdir/main/stconfig.json $outdir
chmod +x $outdir/servicetunnel
cd $outdir
zip -r $zippath .
