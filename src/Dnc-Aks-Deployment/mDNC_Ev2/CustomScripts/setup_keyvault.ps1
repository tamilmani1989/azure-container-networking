param(
    [Parameter(Mandatory = $false)]
    [string]$regionalKeyVaultName = $Env.RegionalKeyVaultName,

    [Parameter(Mandatory = $false)]
    [string]$resourceGroup = $Env.RegionalKeyVaultResourceGroup, 

    [Parameter(Mandatory = $false)]
    [string]$subscriptionId = $Env.SubscriptionId, 

    [Parameter(Mandatory = $false)]
    [string]$location = $Env.location
)

function Create-KeyVault {
    param([string]$resourceGroup)

    Write-Host "Keyvault $regionalKeyVaultName in resource group $resourceGroup not found"
    Write-Host "Creating Keyvault"
    New-AzKeyVault -VaultName $regionalKeyVaultName -ResourceGroupName $resourceGroup -Location $location -Sku 'Premium' -EnabledForDeployment -EnabledForTemplateDeployment -ErrorVariable ProcessError -ErrorAction SilentlyContinue
    if($ProcessError)
   {
     write-host $ProcessError
   }

}

. (Join-Path $PSScriptRoot \common.ps1)

Login-Subscription

$azKeyVault = Get-AzKeyVault -VaultName $regionalKeyVaultName -ResourceGroupName $resourceGroup -ErrorAction SilentlyContinue

if($azKeyVault.ResourceId -eq $null )
{
   $resourceGroupResponse = Get-AzResourceGroup -Name $resourceGroup -Location $location -ErrorAction SilentlyContinue -ErrorVariable ProcessError

   if($ProcessError)
   {
     write-host $ProcessError
   }

   if($resourceGroupResponse.ResourceId -eq $null ){
     Create-ResourceGroup -resourceGroup $resourceGroup
   }

   Create-KeyVault -resourceGroup $resourceGroup

   ## check for secret 
   ## populate secret
      ## read from master keyvault and copy over
} else{
        Write-Host "Keyvault exists, skipping creation"
        $azKeyVault = $azKeyVault | Select *
        Write-Host "$azKeyVault"
}

##this is just a temporary mitigation
## copy certs from a master keyvault into this regions keyvault

# $certificatesJson = Get-Content (Join-Path $PSScriptRoot \service_certificates.json) | ConvertFrom-Json

# foreach($certificate in $certificatesJson.Certificates){
#     $cert = Get-AzKeyVaultCertificate -VaultName $certificatesJson.MasterKeyVault.KeyvaultName -Name $certificate.CertificateName
#     $destcert = Get-AzKeyVaultCertificate -VaultName $env:RegionalKeyVaultName -Name $certificate.CertificateName -ErrorAction SilentlyContinue
#     if(!$destcert)
#     {
#       Write-Host "Update cert: " $certificate.CertificateName
#       $Secret = Get-AzKeyVaultSecret -VaultName $certificatesJson.MasterKeyVault.KeyvaultName -Name $certificate.CertificateName
#       Import-AzKeyVaultCertificate -VaultName $env:RegionalKeyVaultName -CertificateString $Secret.SecretValueText -Name $certificate.CertificateName  
#     }
# }