<#
    .SYNOPSIS
        Deletes cache file if it expired

    .DESCRIPTION
        Deletes cache file if it expired according to cache expiration timeout
	This will lead to do full init and recreate cache

#>

param (
  [string] $CacheFilePath          = $Env:RESTORE_CACHED_ENV_PATH,
  [Int32]  $CacheExpirationTimeout = $Env:INIT_CACHE_EXPIRATION_TIMEOUT
) 

$ErrorActionPreference = "Stop"

if (-not $CacheFilePath) { throw "RESTORE_CACHED_ENV_PATH environment variable is not set" }

if (-not $CacheExpirationTimeout)
{ 
  Write-Host "WRN: INIT_CACHE_EXPIRATION_TIMEOUT environment variable is 0 or not set"

  exit 1
}

$CacheExpirationLimit = (Get-Date).AddHours(-$CacheExpirationTimeout)

if (Test-Path $CacheFilePath)
{
    $CacheFile = Get-Item $CacheFilePath

    if ($CacheFile.LastWriteTime -le $CacheExpirationLimit)
    {
        Write-Host "INF: Deleteing expired cache file $CacheFilePath"

        Remove-item -Path $CacheFilePath -Force
    }
}
