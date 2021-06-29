#!/bin/bash

set -ex

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--cluster-resource-group)
				shift
				cluster_rg="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	run_datapath_tests "$cluster_rg"
}

# Usage: run_datapath_tests <customer cluster rg>
function run_datapath_tests() {
	kubeconfig_secret="$(get_kubeconfig_secret_name "$1")"
	az keyvault secret download -f kubeconfig.json -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	export KUBECONFIG="$(pwd)/kubeconfig.json"
	pushd .
	cd "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && cd ../../../dnc/test/integration && pwd)"
	go test -v . -args --test-kubeconfig="$KUBECONFIG"
	popd
}

main $@