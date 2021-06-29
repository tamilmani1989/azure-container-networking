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

	run_dns_test "$cluster_rg"
}

# Usage: run_dns_test <customer cluster rg>
function run_dns_test() {
	kubeconfig_secret="$(get_kubeconfig_secret_name "$1")"
	az keyvault secret download -f kubeconfig.json -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	export KUBECONFIG="$(pwd)/kubeconfig.json"
	set +e
	for i in $(seq 3); do
		kubectl apply -f "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && cd .. && pwd)"/dnsutils-pod.yaml
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	set -e

	for i in $(seq 3); do
		sleep 30s
		status=$(get_pod_status dnsutils)
		if [[ "$status" == "Running" ]]; then
			break
		fi
	done

	if [[ "$status" != "Running" ]]; then
		return 1
	fi

	set +e
	for i in $(seq 3); do
		kubectl exec -ti dnsutils -- nslookup google.com
		if [[ $? -eq 0 ]]; then
			success="true"
		fi
		sleep 30s
	done
	set -e

	if [[ "$success" == "true" ]]; then
		return 0
	else
		return 1
	fi
}

# Usage: get_pod_status <pod name>
function get_pod_status() {
	set +e
	for i in $(seq 3); do
		sleep 30s
		status=$(kubectl get pod "$1" -ojsonpath="{.status.phase}")
		if [[ $? -eq 0 ]]; then
			echo "$status"
			break
		fi
	done
	set -e
}

main $@