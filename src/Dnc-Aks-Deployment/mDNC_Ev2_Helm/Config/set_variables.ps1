$environmentVariables = gci env:
gci env:

foreach ($Variable in $environmentVariables){
    Write-Host "##vso[task.setvariable variable=$($Variable.Key);]$($Variable.Value)"
}