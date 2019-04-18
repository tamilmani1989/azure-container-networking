## mandatory values
$subscriptionId = ""
$region = "eastus2euap"
$sshPubKeyPath = "C:\Users\tsch\.ssh\id_rsa2\acs-key.pub"
$configurationFilesPath = "C:\Users\tsch\git\goprojects\aquarius_dev\src\Runner\NetworkPerformanceRunner\scripts"
$k8sConfig = "C:\Users\tsch\git\goprojects\aquarius_dev\src\Runner\NetworkPerformanceRunner\scripts\acnLinuxNetPerfMonitoring.json"
$resourceGroup = "runner-notaccelnet"

## constant values
$service = "default/runner-server-svc"

## login and select subscription
Write-Host "Logging in..."
az login
$null = Select-AzureRmSubscription -SubscriptionId $subscriptionId

## create resource group in region
az group create --name $resourceGroup --location $region | jq-win64.exe -r '.properties.provisioningState'
if ($?) {
    ## deploy k8s cluster
    Write-Host "Creating k8s cluster"
    (Get-Content $k8sConfig\kubernetes.json) -replace "`"dnsPrefix`":.*", "`"dnsPrefix`": `"$resourceGroup`"," | Out-File $k8sConfig -Encoding ASCII
    (Get-Content $k8sConfig\kubernetes.json) -replace "`"keyData`":.*", "`"keyData`": `"$(cat $sshPubKeyPath)`"" | Out-File $k8sConfig -Encoding ASCII
    aks-engine.exe generate $k8sConfig
    az group deployment create --name $clusterName --resource-group $resourceGroup --template-file .\_output\$resourceGroup\azuredeploy.json --parameters .\_output\$resourceGroup\azuredeploy.parameters.json

    if ($?) {
        ## retrieve data from k8s cluster
        $k8sEP = az network public-ip list --resource-group $resourceGroup | jq-win64.exe -r '.[0].ipAddress'
        $agentVmName = az vm list --resource-group $resourceGroup | jq-win64.exe -r '.[0].osProfile.computerName'
        $masterVmName = az vm list --resource-group $resourceGroup | jq-win64.exe -r '.[1].osProfile.computerName'

        ## transfer configuration files to master node
        echo yes | scp $configurationFilesPath\runner-server.yaml acn@$k8sEP`:/home/acn

        ## take the following output and replace the script value in configure.json IFF we change commands in configure.sh
        ## [run in bash] cat configure.sh | gzip -9 | base64 -w 0

        ## setup the services by running the configuration scripts
        az vm extension set --resource-group $resourceGroup --vm-name $masterVmName --name customScript --publisher Microsoft.Azure.Extensions --protected-settings $configurationFilesPath\configure-master.json

        ## get the external ip for canary client (may take some time before available...)
        Start-Sleep -s 45
        $lbname = (az network public-ip list --resource-group $resourceGroup | jq-win64.exe --arg service "$service" '.[] | select(.tags.service==$service) | .name').Replace("`"","")
        az network public-ip update -g $resourceGroup -n $lbname --dns-name $resourceGroup.$region`.cloudapp.azure.com --allocation-method Dynamic
    }
}