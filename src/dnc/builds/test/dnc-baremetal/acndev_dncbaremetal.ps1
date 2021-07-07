#builds and pushes the latest dnc-baremetal image
#default folder-path assuming git repo is in current folder
param (
    [string]$folderPath = "Networking-Aquarius"
)
echo $folderPath
cd $folderPath
$commit=(git rev-parse --short HEAD)
$date=Get-Date -Format "MM.dd.yyyy"
$tag = $commit+"_"+$date
docker build -t acndev.azurecr.io/dnc-baremetal:$tag -f src/dnc/builds/dnc/Dockerfile --build-arg GENERATE_BOND_FILE=false .
az acr login --name acndev
docker push acndev.azurecr.io/dnc-baremetal:$tag