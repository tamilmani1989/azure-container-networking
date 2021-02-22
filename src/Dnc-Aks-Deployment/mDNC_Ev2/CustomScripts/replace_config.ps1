$configReplacementJson = Get-Content (Join-Path $PSScriptRoot \service_config_replacement.json) | ConvertFrom-Json

foreach($configReplacement in $configReplacementJson.ServiceConfigReplacement)
{
    $config = Get-Content (Join-Path $PSScriptRoot $configReplacement.SourceConfigFile)
    $destination = Get-Content (Join-Path $PSScriptRoot $configReplacement.DestinationFile)

    ## ev2 overrides on the azure devops task is not respecting escaped json strings
    ## setting the pubsubvips string here
    if($configReplacement.Service -eq 'dnc'){
        $config = $config.replace('__PubsubVips__', $PubsubVips)
    }

    

    $destination.replace('<<config_replace>>', $config) | Set-Content (Join-Path $PSScriptRoot $configReplacement.DestinationFile)
}