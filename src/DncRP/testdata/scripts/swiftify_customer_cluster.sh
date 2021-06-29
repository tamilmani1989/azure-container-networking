#!/bin/bash

set -ex

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--rp-resource-group)
				shift
				rp_resource_group="$1"
				;;
			--cluster-resource-group)
				shift
				cluster_resource_group="$1"
				;;
			--location)
				shift
				location="$1"
				;;
			--subscription)
				shift
				subscription="$1"
				;;
			--service-principal)
				shift
				service_principal="$1"
				;;
			--password)
				shift
				password="$1"
				;;
			--tenant)
				shift
				tenant="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	swiftify_customer_cluster "$rp_resource_group" "$cluster_resource_group" "$location" "$subscription" "$tenant" "$service_principal" "$password"
}

# Usage: swiftify_customer_cluster  <rp resource group> <cluster resource group> <location> <subscription> <tenant> <sp> <password>
function swiftify_customer_cluster() {
	create_resource_group_if_not_exists --resource-group $1 --location "$3"
	point_az_to_region --subscription "$4" --service-principal "$6" --password "$7" --tenant "$5" --region "$3"
	kubeconfig_secret="$(get_kubeconfig_secret_name "$2")"
	az keyvault secret download -f kubeconfig.json -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	orchestrator_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/orchestrators --query '[0].id' --output tsv)
	principal_id=$(az resource show --id "$orchestrator_arm_id" --query "identity.principalId" --output tsv)
	app_id=$(az ad sp show --id $principal_id --query "appId" --output tsv)
	delegated_subnet_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/delegatedSubnets --query '[0].id' --output tsv)
	delegation_guid=$(az resource show --id "$delegated_subnet_arm_id" --query 'properties.resourceGuid' --output tsv)
	sed -i "s/%ORCHESTRATOR_APP_ID%/$app_id/g" "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && cd .. && pwd)"/rbac.yaml
	export KUBECONFIG=kubeconfig.json
	# Sometime kubectl commands can timeout so lets retry a few times
	set +e
	for i in $(seq 3); do
		kubectl label nodes --selector kubernetes.azure.com/role=agent kubernetes.azure.com/podnetwork-delegationguid=$delegation_guid
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	for i in $(seq 3); do
		kubectl label nodes --selector kubernetes.azure.com/role=agent kubernetes.azure.com/podnetwork-subnet=pod-subnet
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	for i in $(seq 3); do
		kubectl apply -f "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && cd .. && pwd)"/rbac.yaml
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	set -e
	# AAD replication seems to take a while, so sleep for 2m here before we consider the cluster ready
	sleep 2m
}

main $@