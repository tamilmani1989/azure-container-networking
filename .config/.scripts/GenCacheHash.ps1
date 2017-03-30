<#
.SYNOPSIS
    Creates cmd file to be used to restore cached init

.DESCRIPTION
    To have builds faster and make sure same packages are used between different build legs,
    we save environment changes made by init script into cmd file to call it instead of init.
#>

param(
  [string] $HashSrcFile    = $Env:ENV_CACHE_HASH_SRC_PATH,
  [string] $HashDestFile   = $Env:ENV_CACHE_HASH_DEST_PATH,

  [string] $CacheLocation  = $Env:INIT_CACHE_ENV_DIR
)

$ErrorActionPreference = "Stop"

if (-not $HashSrcFile)    { throw "ENV_CACHE_HASH_SRC_PATH environment variable is not set" }
if (-not $HashDestFile)   { throw "ENV_CACHE_HASH_DEST_PATH environment variable is not set" }

if (-not $CacheLocation)  { throw "INIT_CACHE_ENV_DIR environment variable is not set" }

$CacheHash = (Get-FileHash -Path $HashSrcFile -Algorithm MD5).Hash

if (Test-Path $HashDestFile)
{
  Remove-Item -Path $HashDestFile -Force
}

Set-Content -Path $HashDestFile -Value $CacheHash




