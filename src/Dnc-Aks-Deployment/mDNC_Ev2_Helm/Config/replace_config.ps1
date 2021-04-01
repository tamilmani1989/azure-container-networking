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

    ## DNC 
    if ($Service -eq "dnc") {
    $config = $config.replace('__PubsubVips__', (Out-String -InputObject (Get-Value "Application" "PubsubVips")).Trim())
    $config = $config.replace('__PubsubCertName__', (Out-String -InputObject (Get-Value "Application" "PubsubCertName")).Trim())
    $config = $config.replace('__PartitionKey__', (Out-String -InputObject (Get-Value "Application" "PartitionKey")).Trim())
    $config = $config.replace('__PubsubConnectionEndpoint__', (Out-String -InputObject (Get-Value "Application" "PubsubConnectionEndpoint")).Trim())
    $config = $config.replace('__MSIResourceID__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())

    $config = $config.replace('__ManagedDncGuid__', (Out-String -InputObject (Get-Value "Application" "ManagedDncGuid")).Trim())
    }

    $config = $config.replace('__CosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDbAccountName")).Trim())
    $config = $config.replace('__CosmosDBResourceID__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDBResourceID")).Trim())
    $config = $config.replace('__RegionalKeyVaultName__', (Out-String -InputObject (Get-Value "Dependencies" "RegionalKeyVaultName")).Trim())

    $config = $config.replace('__MaxBatchSize__', (Out-String -InputObject (Get-Value "Application" "MaxBatchSize" "1000")).Trim())

    ##dncrc
    
    $config = $config.replace('__RpCosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbAccountName")).Trim())
    $config = $config.replace('__RpCosmosDbResoureID__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbResoureID")).Trim())

    $config = $config.replace('__SubscriptionId__', (Out-String -InputObject (Get-Value "Cluster" "SubscriptionId")).Trim())
    $config = $config.replace('__ResourceGroup__', (Out-String -InputObject (Get-Value "Cluster" "ResourceGroup")).Trim())
    $config = $config.replace('__ClusterClientId__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())
    $config = $config.replace('__Location__', (Out-String -InputObject (Get-Value "Cluster" "Location")).Trim())

    ##gateway
    $config = $config.replace('__ClientAppIds__', (Out-String -InputObject (Get-Value "Application" "ClientAppIds")).Trim())

    ## baremetal
    $config = $config.replace('__EnforceNicLImit__', (Out-String -InputObject (Get-Value "Application" "EnforceNicLimit" "true")).Trim())

    # dnc feature flags
    $config = $config.replace('__EnableBatchAllocationProcessing__', (Out-String -InputObject (Get-Value "Application" "EnableBatchAllocationProcessing" "true")).Trim())
    $config = $config.replace('__MaxIPReservationBatchSize__', (Out-String -InputObject (Get-Value "Application" "MaxIPReservationBatchSize" "100")).Trim())
    $config = $config.replace('__UsePrepopulatedIPAllocations__', (Out-String -InputObject (Get-Value "Application" "UsePrepopulatedIPAllocations" "false")).Trim())


    $destinationPath = Join-Path $PSScriptRoot $configReplacement.DestinationFile
    Write-Host "Writing file to $destinationPath"

    $destination = $destination.replace('<<config_replace>>', $config)
    $destination | Set-Content $destinationPath

    Write-Host "$(Get-Content $destinationPath)"

    ## TODO - pick a good destination for these per environment config overrides 
}

