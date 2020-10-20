param(
    [Parameter(Mandatory = $false)]
    [string]$cosmosDbAccountName = $Env.CosmosDbAccountName,

    [Parameter(Mandatory = $false)]
    [string]$resourceGroup = $Env.CosmosDbResourceGroup, 

    [Parameter(Mandatory = $false)]
    [string]$subscriptionId = $Env.SubscriptionId, 

    [Parameter(Mandatory = $false)]
    [string]$location = $Env.location,

    [Parameter(Mandatory = $false)]
    [string]$apiKind = "Table", 

    [Parameter(Mandatory = $false)]
    [string]$consistencyLevel = "Eventual",

    [Parameter(Mandatory = $false)]
    [bool]$skipToSet = $false,

    [Parameter(Mandatory = $false)]
    [string]$outputVariable = "testSetCosmosDbText"
)

function Create-CosmosDB {

   param([string]$resourceGroup)

    Write-Host "CosmosDB $cosmosDbAccountName in resource group $resourceGroup not found"
    Write-Host "Creating CosmosDB"
   
   New-AzCosmosDBAccount `
    -ResourceGroupName $resourceGroup `
    -Location $location `
    -Name $cosmosDbAccountName `
    -ApiKind $apiKind `
    -DefaultConsistencyLevel $consistencyLevel `
    -ErrorAction SilentlyContinue `
    -ErrorVariable ProcessError `

   if($ProcessError)
   {
     write-host $ProcessError
   }
}

. (Join-Path $PSScriptRoot \common.ps1)

Login-Subscription

Install-Module Az.CosmosDB -RequiredVersion 0.1.6 -Scope CurrentUser -Force
Import-Module Az.CosmosDB

$cosmosDB = Get-AzCosmosDBAccount -ResourceGroupName $resourceGroup -Name $cosmosDbAccountName -ErrorAction SilentlyContinue

if(($cosmosDB.Id -eq $null) -and ($skipToSet -eq $false))
{
   $resourceGroupResponse = Get-AzResourceGroup -Name $resourceGroup -Location $location -ErrorAction SilentlyContinue
   if($resourceGroupResponse.ResourceId -eq $null ){
     Create-ResourceGroup -resourceGroup $resourceGroup
   }

   Create-CosmosDB -resourceGroup $resourceGroup
} else{

        ## https://github.com/Microsoft/azure-pipelines-agent/blob/master/docs/preview/outputvariable.md
        $resourceId = $cosmosDB.Id
        Write-Host "CosmosDb exists, skipping creation"
        Write-Host "Setting variable $outputVariable "
        Write-Host "##vso[task.setvariable variable=$outputVariable;isSecret=false;isOutput=true;]$resourceId.Id"
        $cosmosDB = $cosmosDB | Select *
        Write-Host "$cosmosDB"
}





