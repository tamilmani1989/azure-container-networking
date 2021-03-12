param(
    [Parameter(Mandatory = $false)]
    [string]$resourceGroup = $Env.ResourceGroup, 

    [Parameter(Mandatory = $false)]
    [string]$clusterName = $Env.clusterName,
    
    [Parameter(Mandatory = $false)]
    [string]$destinationVariable = $Env.destinationVariable

)


$identityProfile = az aks show --name $clusterName --resource-group $resourceGroup --query  "identityProfile"
$identityProfile = $identityProfile | ConvertFrom-Json

Write-Host $identityProfile

$clusterClientId = $identityProfile.kubeletidentity.clientId
Write-Host "Setting variable ClusterClientId for $destinationVariable to $clusterClientId"        
Write-Host "##vso[task.setvariable variable=$destinationVariable;isSecret=false;isOutput=true;]$clusterClientId"