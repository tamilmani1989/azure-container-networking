##example run
##.\Build-Multiple-DNC.ps1 -InstanceCount 4 -StartingPort 9000 -DestinationDirectory g:\MultipleDnc -SourceDirectory .\

## prereq to using this is to set your dnc config path to the relative to .\config_test.json
##example: setx DNC_CONFIGURATION_PATH .\config_test.json
param(
 [Parameter()]
 [int]$InstanceCount,

 [Parameter()]
 [int]$StartingPort,

 [Parameter()]
 [string]$DestinationDirectory,

 [Parameter()]
 [string]$SourceDirectory
 )


function OverridePort-Config {
 param( [int]$Port, [string]$configFilePath)
    $dncConfigJson = Get-Content $configFilePath | ConvertFrom-Json
    $dncConfigJson.ServerSettings.Port = "" + $Port + ""

    return $dncConfigJson
}

function OutputDnc-Config {
 param([object]$dncConfigJson, [string]$destinationFilePath)

   $dncConfigJson = $dncConfigJson | ConvertTo-Json
   [System.IO.File]::WriteAllLines($destinationFilePath, $dncConfigJson)
}

$endPort = $StartingPort + $InstanceCount 
cd $SourceDirectory

##build each instance and assign it to a port
for($i = $StartingPort; $i -lt $endPort; $i++)
{
    Write-Host "Building DNC with Port $i"
    go build -o azure-dnc-port-$i.exe -ldflags "-X dnc/utils.Version=%VERSION%"

    $exists = Test-Path $DestinationDirectory\$i

    if (!$exists)
    {
        New-Item -ItemType "directory" -Path $DestinationDirectory\$i
    }
    
    ## override config with port 
    $dncConfig = OverridePort-Config -Port $i -configFilePath $SourceDirectory\config_test.json

    ## write output to destination
    OutputDnc-Config -dncConfigJson $dncConfig -destinationFilePath $DestinationDirectory\$i\config_test.json

    Copy-Item azure-dnc-port-$i.exe -Destination $DestinationDirectory\$i\azure-dnc-port-$i.exe
    Copy-Item azure-vnet-telemetry.exe -Destination $DestinationDirectory\$i\azure-vnet-telemetry.exe
    Remove-Item -Path $SourceDirectory\azure-dnc-port-$i.exe
}
