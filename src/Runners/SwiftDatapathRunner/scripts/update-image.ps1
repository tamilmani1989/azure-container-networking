param (
    [parameter(mandatory=$true, HelpMessage="Resource group")]
    [String]
    $resourceGroup,
    [parameter(mandatory=$true, HelpMessage="VM name in resource group (k8s agent node)")]
    [String]
    $vmName
)

az vm extension set --resource-group $resourceGroup --vm-name $vmName --name customScript --publisher Microsoft.Azure.Extensions --settings .\update-image.json
