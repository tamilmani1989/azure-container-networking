<#
.SYNOPSIS
    Create file with current environment deduplicated

.DESCRIPTION
    To reduce environment size and work with it easier we deduplicate it and save to file
#>

param
(
  [Parameter(Mandatory = $true)]
  [string] $Path
)

$ErrorActionPreference = "Stop"

function Dedupe-Environment
{
  param (
    [Parameter(Mandatory = $true)]
    [string] $Name,
    [char] $Delimiter = ';'
  )

  return @(((Get-Item Env:$Name).Value -Split $Delimiter | ? { $_ } | % { Resolve-Path -Path $_ -ErrorAction Ignore } | Select-Object -Unique).Path) -Join $Delimiter
}

function Get-Environment-Deduped
{
  $EnvVarsToDedupe = @("_SHELLPATH", "BUILD_PATH", "PATH", "PATH_0", "PATH_1", "PATH_2")

  Get-ChildItem Env: | % {
    if ($EnvVarsToDedupe -contains $_.Name)
    {
      $_.Value = Dedupe-Environment -Name $_.Name
    }

    Write-Output "@set `"$($_.Name)=$($_.Value)`""
  }
}

Get-Environment-Deduped | Set-Content $Path
