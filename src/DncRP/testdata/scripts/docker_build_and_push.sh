#!/bin/bash

# Builds and pushes one of the following docker images: dnc, dnc-rc, dnc-rp, dnc-gateway
# Usage: build_docker_image.sh --image (dnc|dnc-rc|dnc-rp|dnc-gateway) --tag <tag> --token <token>

set -ex

# source the toolbox
source "$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"/toolbox.sh

function main() {
	# parse the arguments
	while [[ $# -gt 0 ]]; do
		key="$1"
		case $key in
			--image)
				shift
				image="$1"
				;;
			--tag)
				shift
				tag="$1"
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

	docker_build "$image" "$tag" "$token"
	image_repo="$(get_image_for_name "$image")"
	docker_push "$image_repo" "$tag"
}

function docker_push() {
	# image to push is first arg, tag is second
	az acr login -n acndev
	docker push "$1":"$2"
}

function docker_build() {
	# build the image
	case $1 in
		dnc)
			pushd .
			cd ./src/dnc/
			make mdnc-image VERSION=$2
			popd
			;;
		dnc-rc)
			pushd .
			cd ./src/dnc/
			make mdnc-requestcontroller-image VERSION=$2
			popd
			;;
		dnc-rp)
			pushd .
			cd ./src/
			docker build -t acndev.azurecr.io/dncrp:$2 -f DncRP/Dockerfile --no-cache --build-arg FEED_ACCESSTOKEN="$3" .
			popd
			;;
		dnc-gateway)
			pushd .
			cd ./src/
			docker build -t acndev.azurecr.io/dnc-gateway:$2 -f Gateway/Dockerfile --no-cache .
			popd
			;;
	esac
}

main $@