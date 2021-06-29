#!/bin/bash

set -ex

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--force)
				force="true"
				;;
			--cluster-resource-group)
				shift
				cluster_rg="$1"
				;;
			--rp-resource-group)
				shift
				rp_rg="$1"
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

	if [[ "$force" == "true" ]]; then
		cleanup_forced "$rp_rg" "$cluster_rg" "$location" "$subscription" "$tenant" "$service_principal" "$password"
	else
		cleanup_happy_path "$rp_rg" "$cluster_rg" "$location" "$subscription" "$tenant" "$service_principal" "$password"
	fi
}

# Usage cleanup_happy_path <rp resource group> <cluster resource group> <location> <subscription> <tenant> <sp> <password>
function cleanup_happy_path() {
	az extension add --name dnc
	kubeconfig_secret="$(get_kubeconfig_secret_name "$2")"
	az keyvault secret download -f kubeconfig.json -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	availability_set_arm_id=$(az resource list -g "$2" --resource-type Microsoft.Compute/availabilitySets --query "[?starts_with(name, 'agentpool1')]|[0].id" --output tsv)
	az resource delete --ids $(az vm availability-set show --ids "$availability_set_arm_id" --query "virtualMachines[*].id" --output tsv)
	nnc_count=$(kubectl --kubeconfig kubeconfig.json get nnc -n kube-system -ojsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | wc -l)
	while [[ "$nnc_count" -ne 0 ]]; do
		echo "waiting for nnc count to reach 0: got $nnc_count"
		sleep 30s
		nnc_count=$(kubectl --kubeconfig kubeconfig.json get nnc -n kube-system -ojsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' | wc -l)
	done

	az keyvault secret delete -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	set +e
	for i in $(seq 10); do
		sleep 30s
		az keyvault secret purge -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
		if [[ $? -eq 0 ]]; then
			deleted="true"
			break
		fi
	done
	set -e
	if [[ "$deleted" != "true" ]]; then
		echo "unable to purge secret from keyvault"
		return 1
	fi

	point_az_to_region --subscription "$4" --service-principal "$6" --password "$7" --tenant "$5" --region "$3"
                
	delegated_subnet_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/delegatedSubnets --query '[0].id' --output tsv)
	az dnc delegated-subnet-service delete --ids "$delegated_subnet_arm_id" --yes
	orchestrator_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/orchestrators --query '[0].id' --output tsv)
	az dnc orchestrator-instance-service delete --ids "$orchestrator_arm_id" --yes
	controller_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/controller --query '[0].id' --output tsv)
	az dnc controller delete --ids "$controller_arm_id" --yes
	az group delete -g "$1" -y
	az group delete -g "$2" -y
}

# Usage cleanup_forced <rp resource group> <cluster resource group> <location> <subscription> <tenant> <sp> <password>
function cleanup_forced() {
	az extension add --name dnc
	echo "Force deleting RP resources"
	point_az_to_region --subscription "$4" --service-principal "$6" --password "$7" --tenant "$5" --region "$3"
	delegated_subnet_name=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/delegatedSubnets --query '[0].name' --output tsv)
	az dnc delegated-subnet-service delete --force-delete --resource-group "$1" --resource-name "$delegated_subnet_name" --yes
	orchestrator_name=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/orchestrators --query '[0].name' --output tsv)
	az dnc orchestrator-instance-service delete --force-delete --resource-group "$1" --resource-name "$orchestrator_name" --yes
	controller_arm_id=$(az resource list -g "$1" --resource-type Microsoft.DelegatedNetwork/controller --query '[0].id' --output tsv)
	az dnc controller delete --ids "$controller_arm_id" --yes
	az group delete -g "$1" -y
	echo "Deleting customer cluster"
	az group delete -g "$2" --yes
	kubeconfig_secret="$(get_kubeconfig_secret_name "$2")"
	az keyvault secret delete -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
	set +e
	for i in $(seq 10); do
		sleep 30s
		az keyvault secret purge -n "$kubeconfig_secret" --vault-name mdncnightlytestkv
		if [[ $? -eq 0 ]]; then
			deleted="true"
			break
		fi
	done
	set -e
	if [[ "$deleted" != "true" ]]; then
		echo "unable to purge secret from keyvault"
		return 1
	fi
}

main $@