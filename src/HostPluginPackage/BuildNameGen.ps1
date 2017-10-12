param(
    [ValidateNotNullOrEmpty()]
    [string] $BuildFile,
    [string] $BuildType = "amd64fre")

$verXml = [XML](Get-Content (Join-Path $env:VERSIONPATH $env:VERSIONSXMLFILE))
$version = $verXml.root.versions.version.value

$vhdNameFile = Join-Path ([IO.Path]::GetDirectoryName($BuildFile)) "vhdname.txt"
if (Test-Path $vhdNameFile) {
    $vhdName = (Get-Content $vhdNameFile).Trim()
    if ($vhdName -match 'NSM-(.+)-([^-]+)-([\d.]+)-.*(\d{6}-\d{4})-([^.]+)') {
        $repo = $Matches[1]
        $branch = $Matches[2]
        $version = $Matches[3]
        $timestamp = $Matches[4]
        $verhash = $Matches[5]
    }
}
else {
    $repo = "Networking-Aquarius"
    $branch = & git rev-parse --abbrev-ref HEAD
    $timestamp = [DateTime]::Now.ToString("yyMMdd-HHmm")
    $verhash = & git rev-parse --short HEAD
}

"$version.$BuildType.$repo-$branch.$timestamp.$verhash" | Out-File -FilePath $BuildFile -Encoding ascii -Force
