# Copies an image from the default Docker daemon to the Minikube docker daemon
# Usage: push_to_minikube.ps1 source_image destination_image
$sourceImage = $Args[0]
$destinationImage = $Args[1]

docker save -o temp_source_image.tar $sourceImage
& minikube -p minikube docker-env | Invoke-Expression
docker load -i temp_source_image.tar
docker tag $sourceImage $destinationImage
Remove-Item -Force temp_source_image.tar