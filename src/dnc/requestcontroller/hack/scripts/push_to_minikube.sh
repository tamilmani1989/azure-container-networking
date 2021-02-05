#!/bin/bash
# Copies an image from the default Docker daemon to the Minikube docker daemon
# Usage: push_to_minikube.sh source_image destination_image
source_image = $1
destination_image = $2

docker save -o temp_source_image.tar $source_image
eval $(minikube docker-env)
docker load -i temp_source_image.tar
docker tag $source_image $destination_image
rm -f temp_source_image.tar