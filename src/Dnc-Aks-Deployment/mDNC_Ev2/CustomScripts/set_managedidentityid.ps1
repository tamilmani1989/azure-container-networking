param(
    [Parameter(Mandatory = $false)]
    [string]$resourceGroup = $Env.ResourceGroup, 

    [Parameter(Mandatory = $false)]
    [string]$clusterName = $Env.clusterName
)


$identityProfile = az aks show --name $clusterName --resource-group $resourceGroup --query  "identityProfile"
$identityProfile = $identityProfile | ConvertFrom-Json

Write-Host $identityProfile

$clusterClientId = $identityProfile.kubeletidentity.clientId
Write-Host "Setting variable ClusterClientId to $clusterId $clusterClientId"        
Write-Host "##vso[task.setvariable variable=ClusterClientId;isSecret=false;isOutput=true;]$clusterClientId"