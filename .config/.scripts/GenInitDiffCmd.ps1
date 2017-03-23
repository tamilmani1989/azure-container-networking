<#
.SYNOPSIS
    Creates cmd file with environment created by init

.DESCRIPTION
    To have builds faster and make sure same packages are used between different build legs,
    we save environment changes made by init script into cmd file to call it instead of init.
#>

param(
  [string] $PreInitEnvFilePath  = $Env:INIT_CACHE_PRE_ENV_PATH,
  [string] $PostInitEnvFilePath = $Env:INIT_CACHE_POST_ENV_PATH,
  [string] $InitEnvDiffFilePath = $Env:RESTORE_CACHED_ENV_PATH
)

$ErrorActionPreference = "Stop"

if (-not $PreInitEnvFilePath)  { throw "INIT_CACHE_PRE_ENV_PATH environment variable is not set" }
if (-not $PostInitEnvFilePath) { throw "INIT_CACHE_POST_ENV_PATH environment variable is not set" }
if (-not $InitEnvDiffFilePath) { throw "RESTORE_CACHED_ENV_PATH environment variable is not set" }

$PreInitEnvironment  = Get-Content $PreInitEnvFilePath
$PostInitEnvironment = Get-Content $PostInitEnvFilePath

$PreInitEnvPath  = ($PreInitEnvironment  | ? { $_ -imatch "^PATH\=.*$" }) -replace "^PATH\=", ""
$PostInitEnvPath = ($PostInitEnvironment | ? { $_ -imatch "^PATH\=.*$" }) -replace [regex]::escape($PreInitEnvPath), ";%PATH%"

$PostInitEnvironment = $PostInitEnvironment -replace "^PATH\=.*$", $PostInitEnvPath

Compare-Object $PreInitEnvironment $PostInitEnvironment | ? { $_.SideIndicator -eq "=>"} | % { "@set `"$($_.InputObject)`"" } | Set-Content $InitEnvDiffFilePath
