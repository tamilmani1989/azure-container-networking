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

	create_rp_resources "$rp_resource_group" "$cluster_resource_group" "$location" "$subscription" "$tenant" "$service_principal" "$password"
}

# Usage: create_rp_resources <rp resource group> <cluster resource group> <location> <subscription> <tenant> <sp> <password>
function create_rp_resources() {
	kubeconfig_secret="$(get_kubeconfig_secret_name "$2")"
	az keyvault secret download -f kubeconfig.json -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	create_resource_group_if_not_exists --resource-group $1 --location "$3"
	point_az_to_region --subscription "$4" --service-principal "$6" --password "$7" --tenant "$5" --region "$3"
	az extension add --name dnc
	az dnc controller create \
		--location "$3" \
		--resource-group "$1" \
		--resource-name "mdnc-e2e-controller"
	controller_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/controller --query '[0].id' --output tsv)
	ca_data=$(jq -r '.clusters[0].cluster["certificate-authority-data"]' kubeconfig.json)
	apiserver=$(jq -r '.clusters[0].cluster.server' kubeconfig.json)
	private_link_resource_id=$(az network private-link-service show -g "$2" --name ple2etest --query "{id:id}" -o tsv)
	az dnc orchestrator-instance-service create \
		--type "SystemAssigned" \
		--location "$3"  \
		--api-server-endpoint "$apiserver"  \
		--cluster-root-ca "$ca_data" \
		--id "$controller_arm_id" \
		--orchestrator-app-id "4daa1833-1fda-4c06-88b4-d0dcbe8cc169" \
		--orchestrator-tenant-id "$5" \
		--priv-link-resource-id "$private_link_resource_id" \
		--resource-group "$1" \
		--resource-name "mdnc-e2e-cx-cluster"
	cx_vnet_id=$(az resource list -g "$2" --resource-type Microsoft.Network/virtualNetworks --query '[0].id' --output tsv)
	cx_subnet_id=$(az network vnet show --ids $cx_vnet_id --query "subnets[?name=='pod-subnet']|[0].id" --output tsv)
	az dnc delegated-subnet-service create \
		--location "$3" \
		--id "$controller_arm_id" \
		--subnet-details-id "$cx_subnet_id" \
		--resource-group "$1" \
		--resource-name "delegated-k8s-pod-subnet"
}

main $@