#!/bin/bash
set -ex

# Creates a customer cluster using AKS-Engine. In the future, we should migrate to cluster-api

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--resource-group)
				shift
				resource_group="$1"
				;;
			--location)
				shift
				location="$1"
				;;
			--datamodel)
				shift
				datamodel="$1"
				;;
			--subscription)
				shift
				subscription="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	create_customer_cluster "$resource_group" "$location" "$datamodel" "$subscription"
}

# Creates a customer cluster using AKS-E
# Usage create_customer_cluster <resource group> <location> <datamodel> <subscription>
function create_customer_cluster() {
	create_resource_group_if_not_exists --resource-group $1 --location $2
	get_aks_engine
	aks-engine deploy \
		-m "$3" \
		--auth-method cli \
		-l "$2" \
		-s "$4" \
		-g "$1" \
		-p "$1"
	export KUBECONFIG="_output/$1/kubeconfig/kubeconfig.$2.json"
	secret_name=$(get_kubeconfig_secret_name "$1")
	az keyvault secret set --name "$secret_name" --vault-name mdncnightlytestkv --file "$KUBECONFIG" > /dev/null

	vnet_name=$(az resource list -g "$1" --resource-type Microsoft.Network/virtualNetworks --query '[0].name' --output tsv)
	az network vnet subnet update --name "k8s-subnet" --resource-group "$1" --vnet-name "$vnet_name" --disable-private-link-service-network-policies true
	az network lb create --resource-group "$1" --name "ilb-test" --sku Standard --vnet-name "$vnet_name" --subnet "k8s-subnet" --frontend-ip-name feinternal-test --backend-pool-name be-test
	az network lb probe create --resource-group "$1" --lb-name ilb-test --name ilbprobe-test --protocol tcp --port 443
	az network lb rule create --resource-group "$1" --lb-name ilb-test --name ilbrule-test --protocol tcp --frontend-port 443 --backend-port 443 --frontend-ip-name feinternal-test --backend-pool-name be-test --probe-name ilbprobe-test --idle-timeout 15 --enable-tcp-reset true
	
	master_vm_name=$(az vm list -g "$1" --show-details --query "[?contains(name, 'master')].{Name:name}" -o tsv)
	master_vm_nic_resource_id=$(az vm nic list --vm-name "$master_vm_name" -g "$1" --query "[0].id" -o tsv)
	master_vm_nic_name=$(az vm nic show --vm-name "$master_vm_name" -g "$1" --nic "$master_vm_nic_resource_id" --query "{name:name}" -o tsv)

	az network nic ip-config address-pool add --address-pool be-test --ip-config-name "ipconfig1" --nic-name "$master_vm_nic_name" --resource-group "$1" --lb-name ilb-test
	az network private-link-service create -g  "$1" -n ple2etest --vnet-name "$vnet_name" --subnet "k8s-subnet" --lb-name ilb-test --lb-frontend-ip-configs feinternal-test -l "$2"
	az network private-link-service update -g "$1" -n ple2etest --visibility "$4"  --auto-approval "$4" # TODO: Should these be a different param? what if we want to run test in build validations sub
	
	az network vnet subnet create --name "pod-subnet" \
		-g "$1" \
		--vnet-name "$vnet_name" \
		--address-prefixes "10.1.0.0/16" \
		--delegations "Microsoft.DelegatedNetwork/controller"
	
	get_kubectl
	# Sometime kubectl commands can timeout so lets retry a few times
	set +e
	for i in $(seq 3); do
		kubectl apply -f https://raw.githubusercontent.com/Azure/azure-container-networking/master/cns/azure-cns.yaml
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	for i in $(seq 3); do
		kubectl apply -f https://raw.githubusercontent.com/Azure/azure-container-networking/master/tools/acncli/deployment/manager_swift.yaml
		if [[ $? -eq 0 ]]; then
			break
		fi
		sleep 30s
	done
	set -e
}

function get_aks_engine() {
	found=$(which aks-engine || echo "not found")
	if [[ "$found" == "not found" ]]; then
		curl -sSL https://raw.githubusercontent.com/Azure/aks-engine/master/scripts/get-akse.sh | bash
	fi
}

function get_kubectl() {
	found=$(which kubectl || echo "not found")
	if [[ "$found" == "not found" ]]; then
		curl -LO https://storage.googleapis.com/kubernetes-release/release/v1.18.17/bin/linux/amd64/kubectl
		chmod +x ./kubectl
		sudo mv ./kubectl /usr/local/bin/kubectl
	fi
}

main $@