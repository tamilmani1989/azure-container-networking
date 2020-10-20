$configReplacementJson = Get-Content (Join-Path $PSScriptRoot \service_config_replacement.json) | ConvertFrom-Json

foreach($configReplacement in $configReplacementJson.ServiceConfigReplacement)
{
    $config = Get-Content (Join-Path $PSScriptRoot $configReplacement.SourceConfigFile)
    $destination = Get-Content (Join-Path $PSScriptRoot $configReplacement.DestinationFile)
    Write-Host "$config"
    $destination.replace('<<config_replace>>', $config) | Set-Content (Join-Path $PSScriptRoot $configReplacement.DestinationFile)
}
    