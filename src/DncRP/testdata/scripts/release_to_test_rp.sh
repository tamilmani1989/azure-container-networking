#!/bin/bash

set -ex

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--image-tag)
				shift
				image_tag="$1"
				;;
			--token)
				shift
				token="$1"
				;;
			--output-file)
				shift
				output_file="$1"
				;;
			--git-branch)
				shift
				git_branch="$1"
				;;
			*)
				1>&2 echo "unknown argument: $1"
				return 1
				;;
		esac
		shift
	done

	create_ado_release "$image_tag" "$token" "$output_file" "$git_branch"
}

# Creates the release on ADO and saves the response to a file
# Usage create_ado_release <image tag> <access token> <output file> <git branch>
function create_ado_release() {
	git_commit_long=$(git rev-parse HEAD)
	git_commit_short=$(git rev-parse --short HEAD)
	git_commit_title=$(git log --format=%s --max-count=1 $git_commit_long)
	image_tag="$1"

	cat > _release.json <<-EOF
		{
		  "definitionId":30730,
		  "isDraft":false,
		  "description":"Nightly DNC-RP Test",
		  "manualEnvironments":[],
		  "artifacts":[
		    {
		      "alias":"_Networking-Aquarius",
		      "instanceReference": {
		        "id":"$git_commit_long",
		        "name":"$git_commit_short",
		        "sourceBranch":"$4",
		        "commitMessage":"$git_commit_title"
		      }
		    }
		  ],
		  "variables": {
		    "DncBuildTag": {
		      "allowOverride":true,
		      "isSecret":false,
		      "value":"mdnc:$image_tag"
		    },
		    "DncRcBuildTag":{
		      "allowOverride":true,
		      "isSecret":false,
		      "value":"mdnc-requestcontroller:$image_tag"
		    },
		    "DncRpBuildTag":{
		      "allowOverride":true,
		      "isSecret":false,
		      "value":"dncrp:$image_tag"
		    },
		    "GatewayBuildTag":{
		      "allowOverride":true,
		      "isSecret":false,
		      "value":"dnc-gateway:$image_tag"
		    },
		    "MaConfigVersion":{
		      "allowOverride":true,
		      "isSecret":false,
		      "value":"1.7"
		    }
		  },
		  "environmentsMetadata":[
		    {
		      "definitionEnvironmentId":193358,
		      "variables":{}
		    },
		    {
		      "definitionEnvironmentId":193375,
		      "variables":{}
		    }
		  ]
		}
	EOF
	curl -u :"$2" -X POST -d"@_release.json" -H "Content-Type: application/json" 'https://msazure.vsrm.visualstudio.com/One/_apis/Release/releases?api-version=6.0' > "$3"
}

main $@