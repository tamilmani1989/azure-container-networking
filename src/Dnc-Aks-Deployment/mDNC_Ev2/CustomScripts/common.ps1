function Create-ResourceGroup {
    param([string]$resourceGroup)

    Write-Host "Resource group $resourceGroup not found"
    Write-Host "Creating resource group"
    New-AzResourceGroup -Name $resourceGroup -Location $location -Force
}

function Login-Subscription{
    Write-Host "Connecting to azure subscription $subscriptionId"

    $context = Get-AzContext

    if ($context -eq $null) 
    {
        Connect-AzAccount -Subscription $subscriptionId
    } 
    elseif ($context.Subscription.Id -ne $subscriptionId)
    {
        Select-AzureSubscription -SubscriptionId $subscriptionId
    }

    Write-Host "Connected to $subscriptionId"
}