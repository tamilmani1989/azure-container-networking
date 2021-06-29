#!/bin/bash

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--input-file)
				shift
				input_file="$1"
				;;
			--token)
				shift
				token="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	deploy_to_test_rp $input_file $token
}

# Usage deploy_to_test_rp <input file> <token>
function deploy_to_test_rp() {
	release_id=$(jq -r '.id' "$1")
	env_id_1=$(jq -r '.environments[0].id' "$1")
	env_id_2=$(jq -r '.environments[1].id' "$1")
	curl -u :"$2" -X PATCH -d'{"status":2,"scheduledDeploymentTime":null,"comment":""}' -H "Content-Type: application/json" -H "Accept: application/json;api-version=6.1-preview.7" "https://msazure.vsrm.visualstudio.com/One/_apis/Release/releases/$release_id/environments/$env_id_1"
	curl -u :"$2" -X PATCH -d'{"status":2,"scheduledDeploymentTime":null,"comment":""}' -H "Content-Type: application/json" -H "Accept: application/json;api-version=6.1-preview.7" "https://msazure.vsrm.visualstudio.com/One/_apis/Release/releases/$release_id/environments/$env_id_2"
	for i in $(seq 30); do
		curl -u :"$2" https://msazure.vsrm.visualstudio.com/One/_apis/Release/releases/$release_id > release-in-progress.json
		progress_1=$(jq -r '.environments[0].status' release-in-progress.json)
		progress_2=$(jq -r '.environments[1].status' release-in-progress.json)
		if [[ "$progress_1" == "inProgress" ]] || [[ "$progress_2" == "inProgress" ]] || [[ "$progress_1" == "queued" ]] || [[ "$progress_2" == "queued" ]]; then
			sleep 2m
			continue
		else
			break
		fi
	done
	if [[ "$progress_1" == "succeeded" ]] && [[ "$progress_2" == "succeeded" ]]; then
		return
	else
		echo "Deployment failed, expected \"succeeded\" for both envs, but received env1: $progress_1, env2: $progress_2"
		return 1
	fi
}

main $@