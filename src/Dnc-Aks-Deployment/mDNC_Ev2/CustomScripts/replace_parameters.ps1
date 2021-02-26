## replaces values ev2 currently is not replacing, this is to templatize the build out easier
## .\CustomScripts\replace_parameters.ps1 -SourceDirectory D:\ACN_Repo\Networking-Aquarius\src\Dnc-Aks-Deployment\mDNC_Ev2 -subscriptionId "9b8218f9-902a-4d20-a65c-e98acec5362f" -resourceGroup "alegal-test-mdnc-5" -clusterName "alegal-test-mdnc-eastus-5" -location "eastus"
param(
    [Parameter()]
    [string]$SourceDirectory, 

    [Parameter(Mandatory = $false)]
    [string]$subscriptionId = $Env.SubscriptionId, 

    [Parameter(Mandatory = $false)]
    [string]$resourceGroup = $Env.ResourceGroup, 

    [Parameter(Mandatory = $false)]
    [string]$clusterName = $Env.clusterName, 

    [Parameter(Mandatory = $false)]
    [string]$location = $Env.location,

    [Parameter(Mandatory = $false)]
    [string]$releaseName = $Env.releaseName,

    [Parameter(Mandatory = $false)]
    [string]$rolloutParameterFilePath
)


function Override-ServiceModel {
    param(
        [string]$resourceGroup, 
        [string]$configFilePath,
        [string]$subscription,
        [string]$friendlyLocationName,
        [string]$clusterName,
        [string]$rolloutParameterFilePath, 
        [string]$releaseName)

    Write-Host "Updating ServiceModel.json located at $configFilePath"

    $serviceModelJson = Get-Content $configFilePath | ConvertFrom-Json

    $serviceModelJson.serviceResourceGroups[0].azureResourceGroupName = $resourceGroup
    $serviceModelJson.serviceResourceGroups[0].location = $friendlyLocationName
    $serviceModelJson.serviceResourceGroups[0].azureSubscriptionId = $subscription

    $serviceModelJson.applicationDefinitions[0].azureKubernetesService.releaseName = $releaseName

    if($rolloutParameterFilePath){
        $serviceModelJson.serviceResourceGroups[0].serviceResources[0].rolloutParametersPath = $rolloutParameterFilePath
    }

    foreach($application in $serviceModelJson.serviceResourceGroups.serviceResources[0].applications)
    {
          $application.armResourceName = $clusterName
    }

    return $serviceModelJson
}

function Output-File {
    param([object]$configJson, [string]$destinationFilePath)

    
    $configJson = $configJson | ConvertTo-Json -Depth 10

    Write-Host "Writing ServiceModel.json located at $destinationFilePath"
    
    Write-Host "Content ServiceModel.json located at $configJson"
    [System.IO.File]::WriteAllLines($destinationFilePath, $configJson)
}

$serviceModelJson = Override-ServiceModel -configFilePath $SourceDirectory\ServiceModel.json -resourceGroup $resourceGroup -subscription $subscriptionId -friendlyLocationName $location -clusterName $clusterName -rolloutParameterFilePath $rolloutParameterFilePath -releaseName $releaseName


Output-File -configJson $serviceModelJson -destinationFilePath $SourceDirectory\ServiceModel.json