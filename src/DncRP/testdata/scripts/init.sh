#!/bin/bash

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

## This script will generate variables that are used in downstream stages in the pipeline

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			generate-rg-name)
				if [[ "${command}set" == "set" ]]; then
					command="generate-rg-name"
				else
					1>&2 echo "only one of (generate-rg-name, generate-image-tag) must be set"
					return 1
				fi
				;;
			generate-image-tag)
				if [[ "${command}set" == "set" ]]; then
					command="generate-image-tag"
				else
					1>&2 echo "only one of (generate-rg-name, generate-image-tag) must be set"
					return 1
				fi
				;;
			--customer-cluster-1)
				if [[ "${option}set" == "set" ]]; then
					option="customer-cluster-1"
				else
					1>&2 echo "only one of (--customer-cluster-1, --customer-cluster-2, --rp-resources-1, --rp-resources-2) must be set"
					return 1
				fi
				;;
			--customer-cluster-2)
				if [[ "${option}set" == "set" ]]; then
					option="customer-cluster-2"
				else
					1>&2 echo "only one of (--customer-cluster-1, --customer-cluster-2, --rp-resources-1, --rp-resources-2) must be set"
					return 1
				fi
				;;
			--rp-resources-1)
				if [ "${option}set" == "set" ]; then
					option="rp-resources-1"
				else
					1>&2 echo "only one of (--customer-cluster-1, --customer-cluster-2, --rp-resources-1, --rp-resources-2) must be set"
					return 1
				fi
				;;
			--rp-resources-2)
				if [ "${option}set" == "set" ]; then
					option="rp-resources-2"
				else
					1>&2 echo "only one of (--customer-cluster-1, --customer-cluster-2, --rp-resources-1, --rp-resources-2) must be set"
					return 1
				fi
				;;
			--git-commit-short)
				shift
				git_commit="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	case $command in
		generate-rg-name)
			prefix="mdnc-e2e-$(sanitize_string "$git_commit")"
			case $option in
				customer-cluster-1)
					echo "$prefix-cx1"
					;;
				customer-cluster-2)
					echo "$prefix-cx2"
					;;
				rp-resources-1)
					echo "$prefix-rp1"
					;;
				rp-resources-2)
					echo "$prefix-rp2"
					;;
				*)
					1>&2 echo "unknown resource group"
					return 1
					;;
			esac
			;;
		generate-image-tag)
			echo "mdnc-e2e-$(sanitize_string "$git_commit")"
			;;
		*)
			1>&2 echo "unknown subcommand: $command"
			;;
	esac
	
}

main $@