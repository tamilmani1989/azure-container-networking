#!/bin/bash
set -e
version="1.4.0.0"
scriptdir=$(dirname "$0")
outdir=$scriptdir/out/AquariusLinux
zippath="$scriptdir/out/AquariusLinux_$version.zip"
mkdir -p $outdir
cp $scriptdir/scripts/*.sh $outdir
cp $scriptdir/manifest.xml $scriptdir/HandlerManifest.json $outdir
cp $scriptdir/main/servicetunnel $outdir
chmod +x $outdir/*.sh
chmod +x $outdir/servicetunnel
cd $outdir
zip -r $zippath .
