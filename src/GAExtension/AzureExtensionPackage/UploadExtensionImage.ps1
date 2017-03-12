<#
BEFORE RUNNING THIS: Please upload the version mentioned in GAExtensionImage-Aquarius.xml
to the following location:
https://vmnetworkagent.blob.core.windows.net/networkagent/Aquarius.zip
The aboveis in VipReservationTest, download azure storage explorer and use it to upload.
Look at help.PNG file.
#>

Param(
    [Parameter(Mandatory=$false)] [string] $PublishSettingsFile = "C:\Azure\PublishSettingsFile-VipReservationTest.publishsettings",
    [Parameter(Mandatory=$false)] [string] $SubscriptionId = "099b416e-89be-463a-967c-ad4bdec5370e",
    [Parameter(Mandatory=$false)] [string] $definitionFilePath = "C:\Azure\GAExtensionImage-Aquarius.xml",
    [Parameter(Mandatory=$false)] [string] $managementCertThunbprint = 'fa67ef6adc15db0ba94abe2f628e7178e5d76dcc',
    [Parameter(Mandatory=$false)] [string] $version = "0.983"
)


<#  

#1. Create Azure management certificate -> https://azure.microsoft.com/en-us/documentation/articles/cloud-services-certs-create/ 
   $cert = New-SelfSignedCertificate -DnsName sushant-management-cert.azure.net -CertStoreLocation "cert:\LocalMachine\My"
   $password = ConvertTo-SecureString -String 'something' -Force -AsPlainText
   Export-PfxCertificate -Cert $cert -FilePath "C:\Azure\SushantAzureManagementCert.pfx" -Password $password

#2. Uplpload to portal
   $subscriptions = Import-AzurePublishSettingsFile -PublishSettingsFile C:\Azure\PublishSettingsFile-VipReservationTest.publishsettings
   $SubscriptionId = "099b416e-89be-463a-967c-ad4bdec5370e" # Vip reservation Test
   $thumbprint = 'fa67ef6adc15db0ba94abe2f628e7178e5d76dcc'
   $managementCert = Get-Item cert:\\CurrentUser\My\$thumbprint
   Set-AzureSubscription -SubscriptionId $SubscriptionId -Cert $managementCert
   Export-Certificate -Type CERT -Cert $managementCert -FilePath 'C:\Azure\SushantAzureManagementCert.cer'

   # upload certificate to azure using old portal: https://azure.microsoft.com/en-us/documentation/articles/azure-api-management-certs/ 
   # no need to upload certificate to new portal because the extension gets replicated in the background once we upload it via RDFE rest api

#3. Upload package for the first time
    Step 1: Upload the package to the link specified in teh GAExtenionImage-Aquarius.xml
	Step 2: Upload the GAExtensionImage-Aquarius.xml using a rest api

#4. Upload package to upgrade existing package

#>

function PrintCurrentRegisteredPakcages()
{
    # verify current status
    $getUri = "https://management.core.windows.net/" + $selectedSubscriptionId + "/services/publisherextensions"
    $result = Invoke-RestMethod -Method GET -Uri $getUri -Headers @{'x-ms-version'='2014-12-01'} -Certificate $managementCert
    Write-Host "============CURRENT IMAGES============="
    $result.ExtensionImages.ExtensionImage
    Write-Host "============ END CURRENT IMAGES============="
}

$subscriptions = Import-AzurePublishSettingsFile -PublishSettingsFile $PublishSettingsFile

Write-Host "------"
Write-Host "Available Subscriptions:"
foreach($sub in $subscriptions)
{
    Write-Host $sub.Id
}
Write-Host "------"

# by default, this script selects the first subscription
if($subscriptionId -eq "")
{
    Write-Host "Subscription Id not provided. Going to select the first one from Publish Settings."
    selectedSubscriptionId = $subscriptions[0].Id
}
else
{
    $selectedSubscriptionId = $subscriptionId
    Write-Host "Going to use the provided subscription id: $selectedSubscriptionId"
}

Set-AzureSubscription -SubscriptionId $selectedSubscriptionId 

$subscription = Get-AzureSubscription -SubscriptionId $selectedSubscriptionId

$bodyXml = Get-Content $definitionFilePath

$postUri = "https://management.core.windows.net/" + $selectedSubscriptionId + "/services/extensions"

$managementCert = Get-Item cert:\\CurrentUser\My\$managementCertThunbprint

PrintCurrentRegisteredPakcages

# below line is used only for first post
# Write-Host "Uploading the definition file to register a version for the first time"
#Invoke-RestMethod -Method POST -Uri $postUri -Certificate $managementCert -Headers @{'x-ms-version'='2014-12-01'} -Body $bodyxml -ContentType application/xml


Write-Host "Uploading the new definition file to register new version $version"
$updateUri = "https://management.core.windows.net/" + $selectedSubscriptionId + "/services/extensions?action=update"
$headers = New-Object "System.Collections.Generic.Dictionary[[String],[String]]"
$headers.Add('x-ms-version','2014-12-01')
#$headers.Add('Content-Type','application/xml')
Invoke-RestMethod -Method PUT -Uri $updateUri -Certificate $managementCert -Headers $headers -Body $bodyxml -ContentType application/xml

Write-Host "Waiting for 15 seconds after registering the new version before poll"
Start-Sleep -Seconds 15

PrintCurrentRegisteredPakcages

# Check Replication Status

$replicationStatusUri = "https://management.core.windows.net/" + $selectedSubscriptionId + "/services/extensions/Microsoft.Azure.Networking.SDN/Aquarius/" + $version + "/replicationstatus"
$replicationCompleted = $false

$startTime = Get-Date

do
{

    Write-Host "`n`n Going to check replication status..."
    $replicationStatus = Invoke-RestMethod -Method GET -Uri $replicationStatusUri -Headers @{'x-ms-version'='2014-12-01'} -Certificate $managementCert
    $allregions  = $replicationStatus.ReplicationStatusList.ChildNodes
    $replicationCompleted = $true
    #$allregions
    foreach ($region in  $allregions)
    {       
       Write-Host $region.Status $region.Location
    }
    foreach ($region in  $allregions)
    {
        if ($region.Status -eq "NotCompleted")
        {
            $replicationCompleted = $false
            break
        }
    }
    if ($replicationCompleted -eq $false)
    {
        Write-Host "`n`nReplication not yet finished.... will check status again in 10 seconds"
        $currentTime = Get-Date
        $elapsed = (New-TimeSpan -Start $startTime -End $currentTime)
        "Time Elpased since push:" + $elapsed.Hours.ToString() + " hours " + $elapsed.Minutes.ToString() + " minutes " + $elapsed.Seconds.ToString() + " secs"
        Start-Sleep -Seconds 10   
        
    }    

}while($replicationCompleted -eq $false)

$endTime = Get-Date

Write-Host "Replication Started: $startTime -- Replication Ended: $endTime"

# Add extension to IaaS VM 
#$vm = (Get-AzureVM -Name "VMNetworkAgent5" –serviceName "VMNetworkAgent5")
#Get-AzureVMExtension -VM $vm | Select ExtensionName, Publisher, Version
#Set-AzureVMExtension -ExtensionName Aquarius -VM $vm -Publisher Microsoft.Azure.Networking.SDN -Version 0.981 | Update-AzureVM -Verbose
#Get-AzureVMExtension -VM $vm | Select ExtensionName, Publisher, Version

# Add extension to PaaS service
# https://msdn.microsoft.com/en-us/library/azure/dn722510.aspx
<#
Parameter Set: SetExtension
Set-AzureServiceExtension [[-ServiceName] <String> ] [[-Slot] <String> ] [[-Role] <String[]> ] [[-X509Certificate] <X509Certificate2> ] [[-ThumbprintAlgorithm] <String> ] [-ExtensionName] <String> [-ProviderNamespace] <String> [-PublicConfiguration] <String> [-PrivateConfiguration] <String> [-Version] <String> [ <CommonParameters>]

Parameter Set: SetExtensionUsingThumbprint
Set-AzureServiceExtension [[-ServiceName] <String> ] [[-Slot] <String> ] [[-Role] <String[]> ] [-CertificateThumbprint] <String> [[-ThumbprintAlgorithm] <String> ] [-ExtensionName] <String> [-ProviderNamespace] <String> [-PublicConfiguration] <String> [-PrivateConfiguration] <String> [-Version] <String> [ <CommonParameters>]

#>