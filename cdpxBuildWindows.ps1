$GO_VERSION="1.14.2"
$PWD=Get-Location
$DIR=$PWD.Path
$GODIR= $DIR + "\go1-14"
Mkdir $GODIR

$webClient = New-Object System.Net.WebClient; 
$webClient.DownloadFile("https://golang.org/dl/go$GO_VERSION.windows-amd64.zip", "$GODIR\go.zip"); 
cd "$GODIR\"
Write-Host INFO: Expanding go...; 
Expand-Archive $GODIR\go.zip -DestinationPath "$GODIR\"; 

#set env variables
$Env:GOOS = "windows"
$Env:GOARCH = "amd64"
$Env:GOPATH = $DIR
$Env:GOROOT="$GODIR\go"
$Env:PATH = "$GODIR\go\bin;" + $Env:PATH

# Display the environment variables
Get-ChildItem Env:GOOS
Get-ChildItem Env:GOPATH
Get-ChildItem Env:GOROOT
Get-ChildItem Env:PATH

# Check the go version
go version

switch ($args[0])
{
    "bmmodetest" {
        Write-Host $args[0]
        $Env:DNC_CONFIGURATION_PATH = "$DIR\src\dnc\config_test_bmmode.json"
        Write-Host $Env:DNC_CONFIGURATION_PATH
        cd "$DIR\src\dnc\"
        go test -v -tags=bmMode -timeout 2000s -coverprofile coverage-dnc.out
        $EX=$?
        Write-Host INFO: Collecting logs...; 
        Compress-Archive -Path $DIR\src\dnc\azure-dnc* -DestinationPath $DIR\src\dnc\azure-dnc-test-logs.zip
        # Check the status of the last command and properly exit
        if(($? -ne $true) -or ($EX -ne $true)){
          exit 1
        }
    }
}



