# Param(
#     [parameter(mandatory=$true, HelpMessage="Azure subscription id")]
#     [String]
#     $subscriptionId,
#     [parameter(mandatory=$true, HelpMessage="Deployment region")]
#     [String]
#     $region,
#     [parameter(mandatory=$true, HelpMessage="Full path to public ssh key")]
#     [String]
#     $sshPubKeyPath,
#     [parameter(mandatory=$true, HelpMessage="Full path to runner yaml and json files")]
#     [String]
#     $configurationFilesPath,
#     [parameter(mandatory=$true, HelpMessage="Full path to kubernetes json file (for aks-engine)")]
#     [String]
#     $k8sConfig,
#     [parameter(mandatory=$true, HelpMessage="Full path to RunnerDefinition.xml")]
#     [String]
#     $runnerDefinitionXml,
#     [parameter(mandatory=$true, HelpMessage="Full path to RunnerDefinition.xml")]
#     [String]
#     $runnerDefinitionXml
# )
## mandatory values
$subscriptionId = "9bd7ff15-396a-4478-a89b-4d8ae7e302b6"
$region = "westcentralus"
$sshPubKeyPath = "C:\Users\tsch\.ssh\id_rsa2\acs-key.pub"
$configurationFilesPath = "C:\Users\tsch\git\goprojects\aquarius_dev\src\Runners\SwiftDatapathRunner\scripts"
$k8sConfig = "C:\Users\tsch\git\goprojects\aquarius_dev\src\Runners\SwiftDatapathRunner\scripts"
$runnerDefinitionXml = "C:\Users\tsch\git\goprojects\aquarius_dev\src\Runners\SwiftDatapathRunner\client\RunnerDefinition.xml"
$windows = $false

## automated variables (const)
$delegatedSubnet = "delegated"
$resourceGroup = $region + "-sr-rg"
$runnerName = $region + "Runner"
$storageAcc = $region + "st0"
$clusterName = $region + "srCluster0"
$partitionKey = $region + "-sr"
$custaVnetName = "custa-vnet"
$custbVnetName = "custb-vnet"
$custcVnetName = "custc-vnet"
$service = "default/runner-service"
$sharedGWKey = "12d11bfe-9849-4e62-b6f2-bc8af37f7456"

## append W for windows
if ($windows) {
    $resourceGroup += 'W'
    $runnerName += 'W'
    $partitionKey += 'W'
}

## set values in storage.json
$json = "$configurationFilesPath\storage.json"
$content = Get-Content $json | ConvertFrom-json
$content.PartitionKey = $partitionKey
$content | ConvertTo-Json | set-content $json

## set blob and container in runner-conf.yaml
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "([""'].*?)-blob", """$partitionKey-blob".ToLower() | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "([""'].*?)-container", """$partitionKey-container".ToLower() | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

## login and select subscription
Write-Host "Logging in..."
Login-AzureRmAccount
$null = Select-AzureRmSubscription -SubscriptionId $subscriptionId

## set mcr credentials
$pswrd = (az keyvault secret show -n registry-key --vault-name msint --query value).Replace("`"","")
(Get-Content $configurationFilesPath\configure-master.sh) -replace "REPLACEPASSWORD", $pswrd | Out-File $configurationFilesPath\configure-master.sh -Encoding ASCII

## create resource group in region
az group create --name $resourceGroup --location $region | jq-win64.exe -r '.properties.provisioningState'
if ($?) {
    ## deploy k8s cluster
    Write-Host "Creating k8s cluster"
    (Get-Content $k8sConfig\kubernetes.json) -replace "`"dnsPrefix`":.*", "`"dnsPrefix`": `"$resourceGroup`"," | Out-File $k8sConfig\kubernetes.json -Encoding ASCII
    (Get-Content $k8sConfig\kubernetes.json) -replace "`"keyData`":.*", "`"keyData`": `"$(cat $sshPubKeyPath)`"" | Out-File $k8sConfig\kubernetes.json -Encoding ASCII
    aks-engine.exe generate $k8sConfig\kubernetes.json
    az group deployment create --name $clusterName --resource-group $resourceGroup --template-file .\_output\$resourceGroup\azuredeploy.json --parameters .\_output\$resourceGroup\azuredeploy.parameters.json

    if ($?) {
        ## retrieve data from k8s cluster
        $infraVnetName = az network vnet list -g $resourceGroup | jq-win64.exe -r '.[0].name'
        $infraVnet = (az network vnet list -g $resourceGroup | jq-win64.exe --arg infraVnetName "$infraVnetName" '.[] | select(.name==$infraVnetName) | .resourceGuid').Replace("`"","")
        $k8sEP = az network public-ip list --resource-group $resourceGroup | jq-win64.exe -r '.[0].ipAddress'
        $agentVmName = az vm list --resource-group $resourceGroup | jq-win64.exe -r '.[0].osProfile.computerName'
        $masterVmName = az vm list --resource-group $resourceGroup | jq-win64.exe -r '.[1].osProfile.computerName'

        ## save node name
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace """k8s-agentpool1-(.*?"")", """$agentVmName""" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## create customer A vnet; create 'delegated' subnet; create vnet2vnet gateway subnet; and deploy a vm in it
        az network vnet create -g $resourceGroup -n $custaVnetName --address-prefix 10.0.0.0/16 --subnet-name default --subnet-prefix 10.0.0.0/24
        $subnetId = (az network vnet subnet create --address-prefix 10.0.1.0/24 --name $delegatedSubnet --resource-group $resourceGroup --vnet-name $custaVnetName --service-endpoints Microsoft.Storage | jq-win64.exe -r '.id')
        az vm create -n custa-vm1 -g $resourceGroup --image UbuntuLTS --vnet-name $custaVnetName --subnet default --admin-username acn --ssh-key-value $sshPubKeyPath
        $custVnet = (az network vnet list -g $resourceGroup | jq-win64.exe --arg custaVnetName "$custaVnetName" '.[] | select(.name==$custaVnetName) | .resourceGuid').Replace("`"","")
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "(\{){0,1}[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}(\}){0,1}", "$custVnet" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
        az network vnet subnet create --address-prefix 10.0.255.0/27 --name GatewaySubnet --resource-group $resourceGroup --vnet-name $custaVnetName
        az network public-ip create -n custaGWIP -g $resourceGroup --allocation-method Dynamic
        az network vnet-gateway create -n custaGW --public-ip-address custaGWIP -g $resourceGroup --vnet $custaVnetName --gateway-type Vpn --sku VpnGw1 --vpn-type RouteBased --no-wait

        ## create customber B vnet; create gateway subnet; create vnet2vnet gateway subnet; and deploy a vm in it
        az network vnet create -g $resourceGroup -n $custbVnetName --address-prefix 20.0.0.0/16 --subnet-name default --subnet-prefix 20.0.0.0/24
        az network vnet subnet create --address-prefix 20.0.255.0/27 --name GatewaySubnet --resource-group $resourceGroup --vnet-name $custbVnetName
        az network public-ip create -n custbGWIP -g $resourceGroup --allocation-method Dynamic
        az network vnet-gateway create -n custbGW --public-ip-address custbGWIP -g $resourceGroup --vnet $custbVnetName --gateway-type Vpn --sku VpnGw1 --vpn-type RouteBased
        az vm create -n custb-vm1 -g $resourceGroup --image UbuntuLTS --vnet-name $custbVnetName --subnet default --admin-username acn --ssh-key-value $sshPubKeyPath

        ## connect custa and custb vnets
        az network vpn-connection create -n custa2custb -g $resourceGroup --vnet-gateway1 custaGW --shared-key $sharedGWKey --vnet-gateway2 custbGW
        az network vpn-connection create -n custb2custa -g $resourceGroup --vnet-gateway1 custbGW --shared-key $sharedGWKey --vnet-gateway2 custaGW

        ## create custc vnet and VM (for vnet peering test)
        az network vnet create -g $resourceGroup -n $custcVnetName --address-prefix 30.0.0.0/16 --subnet-name default --subnet-prefix 30.0.0.0/24
        az vm create -n custc-vm1 -g $resourceGroup --image UbuntuLTS --vnet-name $custcVnetName --subnet default --admin-username acn --ssh-key-value $sshPubKeyPath
        $peerA = $(az network vnet show --resource-group $resourceGroup --name $custaVnetName --query id --out tsv)
        $peerC = $(az network vnet show --resource-group $resourceGroup --name $custcVnetName --query id --out tsv)
        az network vnet peering create --name custa-custc --resource-group $resourceGroup --vnet-name $custaVnetName --remote-vnet-id $peerC --allow-vnet-access
        az network vnet peering create --name custc-custa --resource-group $resourceGroup --vnet-name $custcVnetName --remote-vnet-id $peerA --allow-vnet-access

        ## create storage endpoint (for service tunnel test)
        az storage account create -n $storageAcc -g $resourceGroup -l $region --sku Standard_LRS
        az storage account network-rule add --resource-group $resourceGroup --account-name $storageAcc --subnet $subnetId
        az storage container create -n swift --account-name $storageAcc --public-access blob
        az storage blob upload -f $configurationFilesPath\testblob.txt -c swift -n testblob.txt --account-name $storageAcc
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "https://.*.blob.core.windows.net/swift/testblob.txt", "https://$storageAcc.blob.core.windows.net/swift/testblob.txt" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## set the vnet so we can delete if we want to
        $virtualNetwork = Get-AzureRmVirtualNetwork -Name $custaVnetName -ResourceGroupName $resourceGroup
        Set-AzureRmVirtualNetwork -VirtualNetwork $virtualNetwork
        $virtualNetwork = Get-AzureRmVirtualNetwork -Name $custbVnetName -ResourceGroupName $resourceGroup
        Set-AzureRmVirtualNetwork -VirtualNetwork $virtualNetwork
        $virtualNetwork = Get-AzureRmVirtualNetwork -Name $custcVnetName -ResourceGroupName $resourceGroup
        Set-AzureRmVirtualNetwork -VirtualNetwork $virtualNetwork
        
        ## SWITCH OVER TO SAW/JIT ACCESS -> swiftify and retrieve authToken using $infraVnet and $custVnet
        ## replace the authToken in subnet.json and continue
        Write-Host "Values for delegation...`nInfraVnet: $infraVnet`nCustVnet: $custVnet`nDncEP: $k8sEP"

        ## transfer configuration files to master node
        echo yes | scp $configurationFilesPath\subnet.json acn@$k8sEP`:/home/acn
        scp $configurationFilesPath\storage.json acn@$k8sEP`:/home/acn
        scp $configurationFilesPath\runner-conf.yaml acn@$k8sEP`:/home/acn

        ## take the following output and replace the script value in configure.json IFF we change commands in configure.sh
        ## [run in bash] cat configure-master.sh | gzip -9 | base64 -w 0
        ## update credentials in configure master script

        ## setup the services by running the configuration scripts
        az vm extension set --resource-group $resourceGroup --vm-name $agentVmName --name customScript --publisher Microsoft.Azure.Extensions --protected-settings $configurationFilesPath\configure-agent.json
        az vm restart --resource-group $resourceGroup --vm-name $agentVmName
        Start-Sleep -s 60
        az vm extension set --resource-group $resourceGroup --vm-name $masterVmName --name customScript --publisher Microsoft.Azure.Extensions --protected-settings $configurationFilesPath\configure-master.json

        ## get the external ip for canary client (may take some time before available...)
        Start-Sleep -s 45
        $canaryEP = (az network public-ip list --resource-group $resourceGroup | jq-win64.exe --arg service "$service" '.[] | select(.tags.service==$service) | .name').Replace("`"","")
        az network public-ip update -g $resourceGroup -n $canaryEP --dns-name swiftrunner --allocation-method Static

        ## set values in RunnerDefinition.xml
        $xml = New-Object xml
        $xml.Load($runnerDefinitionXml)
        $runnerInstances = $xml.SelectNodes("/Runners/RunnerDefinition/Environment/RunnerInstance")
        foreach ($runnerInstance in $RunnerInstances) {
            if ($runnerInstance.RunnerInstanceName.Equals($runnerName)) {
                foreach ($configuration in $runnerInstance.LocalConfigurations.Configuration) {
                    if ($configuration.Name.Equals("K8sEP")) {
                        $configuration.Value = "$k8sEP"
                    } else {
                        $configuration.Value = $partitionKey
                    }
                }

                break
            }
        }
        $xml.Save($runnerDefinitionXml)
    }
}