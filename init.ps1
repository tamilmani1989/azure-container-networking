#Requires -Version 2.0

#
#  Corext Environment Initialization
#
#    Forwards command line arguments to init.cmd and captures environment
#    variable changes to set within the current PowerShell session.
#

[CmdletBinding()]
param
(
    # Use cmdlet binding to support default parameters like -v.
    [Parameter(ValueFromRemainingArguments=$true)]
    [string[]] $Arguments
)

$dp0 = split-path -parent $MyInvocation.MyCommand.Path
$tmp = [System.IO.Path]::GetTempFileName()

write-verbose "Calling `"$dp0\init.cmd`" $Arguments"
cmd /c "call `"$dp0\init.cmd`" $Arguments & set > `"$tmp`""

write-verbose "Parsing variables from `"$tmp`""
get-content $tmp | foreach-object {
    $props = $_ -split '=', 2

    write-verbose "Setting variable `"$($props[0])`": $($props[1])"
    [System.Environment]::SetEnvironmentVariable($props[0], $props[1])
}
