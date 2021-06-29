#!/bin/bash

# Toolbox of bash functions that other scripts can call by sourcing this file

# configure az cli to point to specified region. Should pass in region as subdomain of ARM
# for example, East US 2 EUAP ARM is https://eastus2euap.management.azure.com, so region would be eastus2euap
# arguments:
#  --subscription        Subscription to set as default
#  --service-principle   Service principle id
#  --password            Password for sp
#  --tenant              Tenant ID of sp
#  --region              Region to point to
function point_az_to_region() {
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
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
			--region)
				shift
				region="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	if [[ $(az cloud show --query "name" --output tsv) != "$region" ]]; then
		az cloud register -n $region \
			--endpoint-active-directory https://login.microsoftonline.com \
			--endpoint-active-directory-graph-resource-id https://graph.windows.net/ \
			--endpoint-active-directory-resource-id https://management.core.windows.net/ \
			--endpoint-gallery https://gallery.azure.com/\
			--endpoint-management https://management.core.windows.net/ \
			--endpoint-resource-manager https://$region.management.azure.com/ \
			--suffix-storage-endpoint core.windows.net \
			--suffix-keyvault-dns ".vault.azure.net"
		az cloud set -n $region
		az login --service-principal -u $service_principal -p $password --tenant $tenant
		az account set -s "$subscription"
	fi
}

function create_resource_group_if_not_exists() {
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
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	if [[ "$(az group exists -g "$resource_group")" == "false" ]]; then
		az group create -l "$location" -n "$resource_group"
	fi
}

function get_image_for_name() {
	# name is first arg
	case $1 in
		dnc)
		echo acndev.azurecr.io/mdnc
		;;
	dnc-rc)
		echo acndev.azurecr.io/mdnc-requestcontroller
		;;
	dnc-rp)
		echo acndev.azurecr.io/dncrp
		;;
	dnc-gateway)
		echo acndev.azurecr.io/dnc-gateway
		;;
	*)
		echo unknown
		return 1
		;;
	esac
}

function get_kubeconfig_secret_name() {
	# resource group is first arg
	echo "kubeconfig-$1"
}

# Sanitizes string for use as a resource group name
# Resource group names only allow alphanumeric characters, periods, underscores,
# hyphens and parenthesis and cannot end in a period.
function sanitize_string() {
	dirty_string="$1"
	sanitized_string=$(sed -E 's/[^[:alnum:]\._\-\(\)]+/-/g' <<<"$dirty_string")
	sanitized_string=${sanitized_string%%*(.)}
	echo $sanitized_string
}