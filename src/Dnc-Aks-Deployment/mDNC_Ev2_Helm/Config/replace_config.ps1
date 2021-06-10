param(
    [Parameter()]
    [string]$OverridesPath,
    [Parameter()]
    [string]$Service
)

function Get-Value ([string]$Section, [string]$Key, [string] $defaultValue) {
    if (Test-Path env:$key) {
        write-host "Found environment variable for $key"
        return [System.Environment]::GetEnvironmentVariable($key)
    }
    
    if ($overrides) {
        if (Get-Member -inputobject $overrides.$section -name $key -Membertype Properties) {
            write-host "Found $key in local configuration"
            return $overrides.$section.$key
        }
    }

    if ($defaultValue) {
        return $defaultValue
    }

    throw "[Error] No environment variable or config value for $key found"
}


## TODO - pick a good source for these per environment config overrides 
if ($OverridesPath) {
    $overrides = Get-Content $OverridesPath | ConvertFrom-Json 
}

$configReplacementJson = Get-Content (Join-Path $PSScriptRoot \service_config_replacement.json) | ConvertFrom-Json

foreach ($configReplacement in $configReplacementJson.ServiceConfigReplacement) {
    $config = Get-Content (Join-Path $PSScriptRoot $configReplacement.SourceConfigFile)
    $destination = Get-Content (Join-Path $PSScriptRoot $configReplacement.DestinationFile)

    ## DNC-RP
    if ($Service -eq "dncrp") {
        # RP should skip AME Cert validation in Test Sub
        $config = $config.replace('__SkipAMECertValidation__', (Out-String -InputObject (Get-Value "Application" "SkipAMECertValidation" "false")).Trim())
        $config = $config.replace('__DNCResourceGroup__', (Out-String -InputObject (Get-Value "Application" "DNCResourceGroup", "MC_acn-rg-dnc-centralus_acncluster-dnc-centralus_centralus")).Trim())
        $config = $config.replace('__DNCVnetName__', (Out-String -InputObject (Get-Value "Application" "DNCVnetName", "aks-vnet-20527462")).Trim())
        $config = $config.replace('__DNCPrivateEndpointSubnetName__', (Out-String -InputObject (Get-Value "Application" "DNCPrivateEndpointSubnetName", "privateendpointsubnet")).Trim())
        $config = $config.replace('__TenantId__', (Out-String -InputObject (Get-Value "Cluster" "TenantId", "33e01921-4d64-4f8c-a055-5bdaffd5e33d")).Trim())         
    }

    ## DNC 
    if ($Service -eq "dnc") {
    $config = $config.replace('__PubsubVips__', (Out-String -InputObject (Get-Value "Application" "PubsubVips")).Trim())
    $config = $config.replace('__PubsubCertName__', (Out-String -InputObject (Get-Value "Application" "PubsubCertName")).Trim())
    $config = $config.replace('__EnableRNC__', (Out-String -InputObject (Get-Value "Application" "EnableRNC" "false")).Trim())
    $config = $config.replace('__PartitionKey__', (Out-String -InputObject (Get-Value "Application" "PartitionKey")).Trim())
    $config = $config.replace('__PubsubConnectionEndpoint__', (Out-String -InputObject (Get-Value "Application" "PubsubConnectionEndpoint")).Trim())
    $config = $config.replace('__MSIResourceID__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())
    $config = $config.replace('__ManagedDncGuid__', (Out-String -InputObject (Get-Value "Application" "ManagedDncGuid")).Trim())
    ##gateway
    $config = $config.replace('__ClientAppIds__', (Out-String -InputObject (Get-Value "Application" "ClientAppIds")).Trim())
    }
    
    ##macinventorymanager or macallactor condition
    if (($Service -eq "macinventory") -or ($Service -eq "macallocator")) {
        $config = $config.replace('__CosmosDbSqlAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDbSqlAccountName")).Trim())
    }
    
    $config = $config.replace('__CosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDbAccountName")).Trim())
    $config = $config.replace('__CosmosDBResourceID__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDBResourceID")).Trim())
    $config = $config.replace('__RegionalKeyVaultName__', (Out-String -InputObject (Get-Value "Dependencies" "RegionalKeyVaultName")).Trim())

    $config = $config.replace('__MaxBatchSize__', (Out-String -InputObject (Get-Value "Application" "MaxBatchSize" "1000")).Trim())

    ##dncrc
    
    $config = $config.replace('__RpCosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbAccountName")).Trim())
    $config = $config.replace('__RpCosmosDbResoureID__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbResoureID")).Trim())
    $config = $config.replace('__BatchSize__', (Out-String -InputObject (Get-Value "Application" "ScalerBatchSize" "30")).Trim())
    $config = $config.replace('__RequestThresholdPercent__', (Out-String -InputObject (Get-Value "Application" "RequestThresholdPercent" "50")).Trim())
    $config = $config.replace('__ReleaseThresholdPercent__', (Out-String -InputObject (Get-Value "Application" "ReleaseThresholdPercent" "150")).Trim())

    $config = $config.replace('__SubscriptionId__', (Out-String -InputObject (Get-Value "Cluster" "SubscriptionId")).Trim())
    $config = $config.replace('__ResourceGroup__', (Out-String -InputObject (Get-Value "Cluster" "ResourceGroup")).Trim())
    $config = $config.replace('__ClusterClientId__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())
    $config = $config.replace('__Location__', (Out-String -InputObject (Get-Value "Cluster" "Location")).Trim())

    ## baremetal
    $config = $config.replace('__EnforceNicLImit__', (Out-String -InputObject (Get-Value "Application" "EnforceNicLimit" "true")).Trim())

    # dnc feature flags
    $config = $config.replace('__EnableBatchAllocationProcessing__', (Out-String -InputObject (Get-Value "Application" "EnableBatchAllocationProcessing" "true")).Trim())
    $config = $config.replace('__MaxIPReservationBatchSize__', (Out-String -InputObject (Get-Value "Application" "MaxIPReservationBatchSize" "100")).Trim())
    $config = $config.replace('__UsePrepopulatedIPAllocations__', (Out-String -InputObject (Get-Value "Application" "UsePrepopulatedIPAllocations" "true")).Trim())

    $destinationPath = Join-Path $PSScriptRoot $configReplacement.DestinationFile
    Write-Host "Writing file to $destinationPath"
    
    $destination = $destination.replace('<<config_replace>>', $config)
    $destination | Set-Content $destinationPath

    Write-Host "$(Get-Content $destinationPath)"

    ## TODO - pick a good destination for these per environment config overrides 
}

