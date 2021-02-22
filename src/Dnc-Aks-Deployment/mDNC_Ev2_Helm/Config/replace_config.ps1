param(
    [Parameter()]
    [string]$OverridesPath
)

function Get-Value ([string]$Section, [string]$Key) {
    if (Test-Path env:$key) {
        write-host "Found environment variable for $key"
        return env:$key
    }
    if ($overrides) {
        if (Get-Member -inputobject $overrides.$section -name $key -Membertype Properties) {
            write-host "Found $key in local configuration"
            return $overrides.$section.$key
        }
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
    if ($configReplacement.Service -eq "dnc") {
    $config = $config.replace('__PubsubVips__', (Out-String -InputObject (Get-Value "Application" "PubsubVips")).Trim())
    $config = $config.replace('__PubsubCertName__', (Out-String -InputObject (Get-Value "Application" "PubsubCertName")).Trim())
    $config = $config.replace('__PartitionKey__', (Out-String -InputObject (Get-Value "Application" "PartitionKey")).Trim())
    $config = $config.replace('__PubsubConnectionEndpoint__', (Out-String -InputObject (Get-Value "Application" "PubsubConnectionEndpoint")).Trim())
    $config = $config.replace('__MSIResourceID__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())
    }

    $config = $config.replace('__CosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDbAccountName")).Trim())
    $config = $config.replace('__CosmosDBResourceID__', (Out-String -InputObject (Get-Value "Dependencies" "CosmosDBResourceID")).Trim())
    $config = $config.replace('__RegionalKeyVaultName__', (Out-String -InputObject (Get-Value "Dependencies" "RegionalKeyVaultName")).Trim())

    ##dncrc
    $config = $config.replace('__ManagedDncGuid__', (Out-String -InputObject (Get-Value "Application" "ManagedDncGuid")).Trim())
    
    $config = $config.replace('__RpCosmosDbAccountName__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbAccountName")).Trim())
    $config = $config.replace('__RpCosmosDbResoureID__', (Out-String -InputObject (Get-Value "Dependencies" "RpCosmosDbResoureID")).Trim())

    ##dncrp 
    if ($configReplacement.Service -eq "dncrp") {
        $config = $config.replace('__KeyVaultCertPath__', (Out-String -InputObject (Get-Value "Application" "KeyVaultCertPath")).Trim())
    }
    

    $config = $config.replace('__SubscriptionId__', (Out-String -InputObject (Get-Value "Cluster" "SubscriptionId")).Trim())
    $config = $config.replace('__ResourceGroup__', (Out-String -InputObject (Get-Value "Cluster" "ResourceGroup")).Trim())
    $config = $config.replace('__ClusterClientId__', (Out-String -InputObject (Get-Value "Cluster" "ClusterClientId")).Trim())
    $config = $config.replace('__Location__', (Out-String -InputObject (Get-Value "Cluster" "Location")).Trim())

    $config = $config.replace('__Location__', (Out-String -InputObject (Get-Value "Cluster" "Location")).Trim())


    ##gateway
    $destinationPath = Join-Path $PSScriptRoot $configReplacement.DestinationFile
    Write-Host "Writing file to $destinationPath"

    $destination = $destination.replace('<<config_replace>>', $config)
    $destination | Set-Content $destinationPath

    Write-Host "$(Get-Content $destinationPath)"

    ## TODO - pick a good destination for these per environment config overrides 
}

