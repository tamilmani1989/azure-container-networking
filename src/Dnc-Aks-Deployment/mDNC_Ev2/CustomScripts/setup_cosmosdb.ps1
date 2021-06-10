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
    [string]$outputVariable = "testSetCosmosDbText",

    [Parameter(Mandatory = $false)]
    [string]$createSql = $Env.CreateSql
)

function Create-CosmosDB {
  param([string]$resourceGroup)

  Write-Host "Creating CosmosDB location obj"

  $locationObject = New-AzCosmosDBLocationObject -LocationName $location -FailoverPriority 0 -IsZoneRedundant 1
    Write-Host "CosmosDB $cosmosDbAccountName in resource group $resourceGroup not found"
    Write-Host "Creating CosmosDB"

  New-AzCosmosDBAccount `
  -ResourceGroupName $resourceGroup `
  -LocationObject $locationObject `
  -Name $cosmosDbAccountName `
  -EnableAutomaticFailover `
  -ApiKind $apiKind `
  -DefaultConsistencyLevel $consistencyLevel `
  -ErrorAction SilentlyContinue `
  -ErrorVariable ProcessError `

  if($ProcessError)
  {
    write-host $ProcessError
  }
}

function Create-Sql-CosmosDB {
    
   param([string]$resourceGroup)

    Write-Host "CosmosDB $cosmosDbSqlAccountName in resource group $resourceGroup not found"
    Write-Host "Creating CosmosDB SQL"
   
   New-AzCosmosDBAccount `
    -ResourceGroupName $resourceGroup `
    -Location $location `
    -Name $cosmosDbSqlAccountName `
    -ApiKind "Sql" `
    -DefaultConsistencyLevel $consistencyLevel `
    -ErrorAction SilentlyContinue `
    -ErrorVariable ProcessError `

   if($ProcessError)
   {
     write-host $ProcessError
   }
}
. (Join-Path $PSScriptRoot \common.ps1)

Write-Host "Login subscription"
Login-Subscription

Install-Module Az.CosmosDB -RequiredVersion 0.1.6 -Scope CurrentUser -Force
Import-Module Az.CosmosDB

Write-Host "Check Cosmos Resource Group Exist"
$resourceGroupResponse = Get-AzResourceGroup -Name $resourceGroup -Location $location -ErrorAction SilentlyContinue
if($resourceGroupResponse.ResourceId -eq $null ){
  Write-Host "Create Cosmos Resource Group $resourceGroup"
  Create-ResourceGroup -resourceGroup $resourceGroup
}


Write-Host "Create Cosmos Account"
$cosmosDB = Get-AzCosmosDBAccount -ResourceGroupName $resourceGroup -Name $cosmosDbAccountName -ErrorAction SilentlyContinue
if($cosmosDB.Id -eq $null)
{
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

if($createSql -eq "true"){
    $resourceGroupResponse = Get-AzResourceGroup -Name $resourceGroup -Location $location
    if($resourceGroupResponse.ResourceId -eq $null ){
      Create-ResourceGroup -resourceGroup $resourceGroup
    }

    $cosmosDbSqlAccountName = $cosmosDbAccountName+"-sql"
    $cosmosDBSql = Get-AzCosmosDBAccount -ResourceGroupName $resourceGroup -Name $cosmosDbSqlAccountName -ErrorAction SilentlyContinue
     
    if($cosmosDBSql.Id -eq $null)
    {
      Create-Sql-CosmosDB -resourceGroup $resourceGroup
    } else {

        ## https://github.com/Microsoft/azure-pipelines-agent/blob/master/docs/preview/outputvariable.md
        $resourceId = $cosmosDBSql.Id
        Write-Host "CosmosDbSql exists, skipping creation"
        Write-Host "Setting variable $outputVariable "
        Write-Host "##vso[task.setvariable variable=$outputVariable;isSecret=false;isOutput=true;]$resourceId.Id"
        $cosmosDBSql = $cosmosDBSql | Select *
        Write-Host "$cosmosDBSql"
    }
}
