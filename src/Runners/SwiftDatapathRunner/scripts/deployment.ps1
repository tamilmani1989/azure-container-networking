#find the aquarius gopath
$gopath = $env:GOPATH -split ';'
foreach ($path in $gopath) {
    if ($path -Match "Aquarius") {
        $gopath = $path;
    }
}

# cd to the aquarius gopath directory
$runnerpath = $gopath + "\src\Runners\SwiftDatapathRunner"
$configurationFilesPath = $runnerpath + "\scripts"

## mandatory values
$storageKey = ####
$subscriptionId = "9bd7ff15-396a-4478-a89b-4d8ae7e302b6"
$region = "eastus2euap"
$sshPubKeyPath = "C:\Users\tsch\.ssh\id_rsa2\acs-key.pub"
$runnerDefinitionXml = "$runnerpath\client\RunnerDefinition.xml"
$windows = $false
$canary = $true

## automated variables (const)
$delegatedSubnet = "delegated"
$plSubnet = "privateLinkSubnet"
$resourceGroup = $region + "-sr-rg"
$runnerName = $region + "Runner"
$storageAcc = $region + "st0"
$plStorageAcc = $region + "plst0"
$clusterName = $region + "srCluster0"
$partitionKey = $region + "-sr"
$custaVnetName = "custa-vnet"
$custbVnetName = "custb-vnet"
$custcVnetName = "custc-vnet"
$custdVnetName = "custd-vnet"
$custVnetWithMNAT = "cust-mnat"
$custVnetWithLB = "cust-lb"
$service = "default/runner-service"
$sharedGWKey = "12d11bfe-9849-4e62-b6f2-bc8af37f7456"

## append W for windows
if ($windows) {
    $resourceGroup += 'W'
    $runnerName += 'W'
    $partitionKey += 'W'
}

## set storage key, partition key, blob and container in runner-conf.yaml
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_STORAGE_KEY", $storageKey | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "([""'].*?)-partition", """$partitionKey-partition".ToLower() | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "([""'].*?)-blob", """$partitionKey-blob".ToLower() | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
(Get-Content $configurationFilesPath\runner-conf.yaml) -replace "([""'].*?)-container", """$partitionKey-container".ToLower() | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

## login and select subscription
Write-Host "Logging in..."
Login-AzureRmAccount
$null = Select-AzureRmSubscription -SubscriptionId $subscriptionId

## login and select subscription for az cli
az login
az account set --subscription $subscriptionId

## set mcr credentials
$appid = (az keyvault secret show -n registry-appid --vault-name msint-community --query value).Replace("`"","")
$pswrd = (az keyvault secret show -n registry-key --vault-name msint-community --query value).Replace("`"","")
(Get-Content $configurationFilesPath\configure-master.sh) -replace "REPLACEPASSWORD", $pswrd | Out-File $configurationFilesPath\configure-master.sh -Encoding ASCII

## create resource group in region
az group create -n $resourceGroup -l $region | jq.exe -r '.properties.provisioningState'
if ($?) {
    ## deploy k8s cluster
    Write-Host "Creating k8s cluster"
    (Get-Content $configurationFilesPath\kubernetes.json) -replace "`"dnsPrefix`":.*", "`"dnsPrefix`": `"$resourceGroup`"," | Out-File $configurationFilesPath\kubernetes.json -Encoding ASCII
    (Get-Content $configurationFilesPath\kubernetes.json) -replace "`"keyData`":.*", "`"keyData`": `"$(cat $sshPubKeyPath)`"" | Out-File $configurationFilesPath\kubernetes.json -Encoding ASCII
    # use aks-engine in prod and aks in canary
    aks-engine.exe generate $configurationFilesPath\kubernetes.json
    az group deployment create `
        -n $clusterName `
        -g $resourceGroup `
        --template-file .\_output\$resourceGroup\azuredeploy.json `
        --parameters .\_output\$resourceGroup\azuredeploy.parameters.json

    if ($?) {
        ## retrieve data from k8s cluster
        $infraVnetName = az network vnet list -g $resourceGroup | jq.exe -r '.[0].name'
        $infraVnet = (az network vnet list -g $resourceGroup | jq.exe --arg infraVnetName "$infraVnetName" '.[] | select(.name==$infraVnetName) | .resourceGuid').Replace("`"","")
        $k8sEP = az network public-ip list -g $resourceGroup | jq.exe -r '.[0].ipAddress'
        $agentVm1Name = az vm list -g $resourceGroup | jq.exe -r '.[0].osProfile.computerName'
        $agentVm2Name = az vm list -g $resourceGroup | jq.exe -r '.[1].osProfile.computerName'
        $masterVmName = az vm list -g $resourceGroup | jq.exe -r '.[2].osProfile.computerName'
        if ($agentVm1Name.Contains("master")) {
            $temp = $agentVm1Name
            $agentVm1Name = $masterVmName
            $masterVmName = $temp
        } elseif ($agentVm2Name.Contains("master")) {
            $temp = $agentVm2Name
            $agentVm2Name = $masterVmName
            $masterVmName = $temp
        }

        ## save node names
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace """k8s-agentpool1-(.*?)-0""", """$agentVm1Name""" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace """k8s-agentpool1-(.*?)-1""", """$agentVm2Name""" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
        (Get-Content $configurationFilesPath\configure-master.sh) -replace "NODE-1", $agentVm1Name | Out-File $configurationFilesPath\configure-master.sh -Encoding ASCII
        (Get-Content $configurationFilesPath\configure-master.sh) -replace "NODE-2", $agentVm2Name | Out-File $configurationFilesPath\configure-master.sh -Encoding ASCII

        ## create customer A vnet; create 'delegated' subnet; and deploy 2 VMs in it
        az network vnet create -g $resourceGroup `
            -n $custaVnetName `
            --address-prefix 10.0.0.0/16 `
            --subnet-name default `
            --subnet-prefix 10.0.0.0/24
        $subnetId = (az network vnet subnet create `
                        --address-prefix 10.0.1.0/24 `
                        -n $delegatedSubnet `
                        -g $resourceGroup `
                        --vnet-name $custaVnetName `
                        --service-endpoints Microsoft.Storage | jq.exe -r '.id')
        $vm1Nic = (az vm create `
            -n custa-vm1 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custaVnetName `
            --subnet default `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath | jq.exe -r '.networkProfile.networkInterfaces[0].id').Split('/')[8]
        az vm create `
            -n custa-vm2 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custaVnetName `
            --subnet default `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath
        $custVnet = (az network vnet list -g $resourceGroup | jq.exe --arg custaVnetName "$custaVnetName" '.[] | select(.name==$custaVnetName) | .resourceGuid').Replace("`"","")
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "(\{){0,1}[0-9a-fA-F]{8}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{4}\-[0-9a-fA-F]{12}(\}){0,1}", "$custVnet" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII
        
        ## create vnet2vnet subnet and gateway
        az network vnet subnet create `
            --address-prefix 10.0.255.0/27 `
            -n GatewaySubnet `
            -g $resourceGroup `
            --vnet-name $custaVnetName
        az network public-ip create `
            -n custaGWIP `
            -g $resourceGroup `
            --allocation-method Dynamic
        az network vnet-gateway create `
            -n custaGW `
            --public-ip-address custaGWIP `
            -g $resourceGroup `
            --vnet $custaVnetName `
            --gateway-type Vpn `
            --sku VpnGw1 `
            --vpn-type RouteBased `
            --no-wait
        
        ## create Private Link subnet; storage resource; disable network policies on subnet; configure private dns zone
        az network vnet subnet create `
            --address-prefix 10.0.2.0/24 `
            -n $plSubnet `
            -g $resourceGroup `
            --vnet-name $custaVnetName
        az network vnet subnet update `
            -n $plSubnet `
            -g $resourceGroup `
            --vnet-name $custaVnetName `
            --disable-private-endpoint-network-policies true
        az vm create `
            -n privateLink-vm1 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custaVnetName `
            --subnet $plSubnet `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath
        az storage account create `
            -n $plStorageAcc `
            -g $resourceGroup `
            -l $region `
            --sku Standard_LRS `
            --kind StorageV2 `
            --https-only false `
            --default-action Allow
        az storage account network-rule add `
            -g $resourceGroup `
            --account-name $plStorageAcc `
            --vnet-name $custaVnetName `
            --subnet $delegatedSubnet
        az storage container create `
            -n swiftpl `
            --account-name $plStorageAcc `
            --public-access container
        az storage blob upload `
            -f $configurationFilesPath\testblob.txt `
            -c swiftpl `
            -n testblob.txt `
            --account-name $plStorageAcc
        az storage account update `
            -n $plStorageAcc `
            -g $resourceGroup `
            --default-action Deny
        $privateResourceId = (az storage account show -n $plStorageAcc -g $resourceGroup --query 'id')
        az network private-endpoint create `
            -n myPrivateEndpoint `
            -g $resourceGroup `
            --vnet-name $custaVnetName `
            --subnet $plSubnet `
            --private-connection-resource-id $privateResourceId `
            --connection-name myConnection `
            --group-ids blob
        $ifaceId = (az network private-endpoint show `
                        -n myPrivateEndpoint `
                        -g $resourceGroup `
                        --query 'networkInterfaces[0].id')
        $privateIp = (az resource show --ids $ifaceId --api-version 2019-04-01 -o json | jq.exe -r '.properties.ipConfigurations[0].properties.privateIPAddress')
        az network private-dns zone create `
            -n "privatelink.blob.core.windows.net" `
            -g $resourceGroup
        az network private-dns link vnet create `
            -n myDnsLink `
            -g $resourceGroup `
            -z "privatelink.blob.core.windows.net" `
            -v $custaVnetName `
            --registration-enabled false
        az network private-dns record-set a create `
            -n $plStorageAcc `
            -z privatelink.blob.core.windows.net `
            -g $resourceGroup
        az network private-dns record-set a add-record `
            -n $plStorageAcc `
            -z privatelink.blob.core.windows.net `
            -g $resourceGroup `
            -a $privateIp
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "https://.*.blob.core.windows.net/swiftpl/testblob.txt", "https://$plStorageAcc.blob.core.windows.net/swiftpl/testblob.txt" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## create UDR (testing NC->Next-Hop VM->Destination VM)
        az network route-table create `
            -g $resourceGroup `
            -n udrt
        az network route-table route create `
            -g $resourceGroup `
            --route-table-name udrt `
            -n route1 `
            --next-hop-type VirtualAppliance `
            --address-prefix 10.0.0.5/32 `
            --next-hop-ip-address 10.0.1.5
        az network vnet subnet update `
            --vnet-name $custaVnetName `
            -n $delegatedSubnet `
            -g $resourceGroup `
            --route-table udrt
        # az network nic update `
        #     -n $vm1Nic `
        #     -g $resourceGroup `
        #     --ip-forwarding true
        # az vm extension set `
        #     -g $resourceGroup `
        #     --vm-name custa-vm1 `
        #     -n customScript `
        #     --publisher Microsoft.Azure.Extensions `
        #     --settings $configurationFilesPath/enableFwd.json

        ## create customber B vnet; create gateway subnet; create vnet2vnet gateway subnet; and deploy a vm in it
        az network vnet create `
            -g $resourceGroup `
            -n $custbVnetName `
            --address-prefix 20.0.0.0/16 `
            --subnet-name default `
            --subnet-prefix 20.0.0.0/24
        az network vnet subnet create `
            --address-prefix 20.0.255.0/27 `
            -n GatewaySubnet `
            -g $resourceGroup `
            --vnet-name $custbVnetName
        az network public-ip create `
            -n custbGWIP `
            -g $resourceGroup `
            --allocation-method Dynamic
        az network vnet-gateway create `
            -n custbGW `
            --public-ip-address custbGWIP `
            -g $resourceGroup `
            --vnet $custbVnetName `
            --gateway-type Vpn `
            --sku VpnGw1 `
            --vpn-type RouteBased
        az vm create -n custb-vm1 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custbVnetName `
            --subnet default `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath

        ## connect custa and custb vnets (for GW tests)
        az network vpn-connection create `
            -n custa2custb `
            -g $resourceGroup `
            --vnet-gateway1 custaGW `
            --shared-key $sharedGWKey `
            --vnet-gateway2 custbGW
        az network vpn-connection create `
            -n custb2custa `
            -g $resourceGroup `
            --vnet-gateway1 custbGW `
            --shared-key $sharedGWKey `
            --vnet-gateway2 custaGW

        ## create custc vnet and VM (RFC1918 peering)
        az network vnet create `
            -g $resourceGroup `
            -n $custcVnetName `
            --address-prefix 172.16.0.0/16 `
            --subnet-name default `
            --subnet-prefix 172.16.0.0/24
        az vm create `
            -n custc-vm1 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custcVnetName `
            --subnet default `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath
        $peerA = $(az network vnet show `
                    -g $resourceGroup `
                    -n $custaVnetName `
                    --query id `
                    --out tsv)
        $peerC = $(az network vnet show `
                    -g $resourceGroup `
                    -n $custcVnetName `
                    --query id `
                    --out tsv)
        az network vnet peering create `
            -n custa-custc `
            -g $resourceGroup `
            --vnet-name $custaVnetName `
            --remote-vnet $peerC `
            --allow-vnet-access
        az network vnet peering create `
            -n custc-custa `
            -g $resourceGroup `
            --vnet-name $custcVnetName `
            --remote-vnet $peerA `
            --allow-vnet-access

        ## create custd vnet and VM (Non-RFC1918 peering)
        az network vnet create `
            -g $resourceGroup `
            -n $custdVnetName `
            --address-prefix 30.0.0.0/16 `
            --subnet-name default `
            --subnet-prefix 30.0.0.0/24
        az vm create `
            -n custd-vm1 `
            -g $resourceGroup `
            --image UbuntuLTS `
            --vnet-name $custdVnetName `
            --subnet default `
            --admin-username acn `
            --ssh-key-value $sshPubKeyPath
        $peerD = $(az network vnet show `
                    -g $resourceGroup `
                    -n $custdVnetName `
                    --query id `
                    --out tsv)
        az network vnet peering create `
            -n custa-custd `
            -g $resourceGroup `
            --vnet-name $custaVnetName `
            --remote-vnet $peerD `
            --allow-vnet-access
        az network vnet peering create `
            -n custd-custa `
            -g $resourceGroup `
            --vnet-name $custdVnetName `
            --remote-vnet $peerA `
            --allow-vnet-access

        ## create storage endpoint (for service tunnel test)
        az storage account create `
            -n $storageAcc `
            -g $resourceGroup `
            -l $region `
            --sku Standard_LRS `
            --default-action Allow
        az storage account network-rule add `
            -g $resourceGroup `
            --account-name $storageAcc `
            --subnet $subnetId
        az storage container create `
            -n swift `
            --account-name $storageAcc `
            --public-access container
        az storage blob upload `
            -f $configurationFilesPath\testblob.txt `
            -c swift `
            -n testblob.txt `
            --account-name $storageAcc
        az storage account update `
            -n $storageAcc `
            -g $resourceGroup `
            --default-action Deny
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "https://.*.blob.core.windows.net/swift/testblob.txt", "https://$storageAcc.blob.core.windows.net/swift/testblob.txt" | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## create NSG
        az network nsg create `
            -g $resourceGroup `
            -n myNsg `
            --tags e2e_nsg
        az network nsg rule create `
            -g $resourceGroup `
            -n denyPingFromPod2 `
            --nsg-name myNsg `
            --priority 400 `
            --source-address-prefixes 10.0.1.5/32 `
            --destination-address-prefixes 10.0.0.5/32 `
            --access deny `
            --protocol Icmp `
            --destination-port-ranges * `
            --description "Deny pod2 to ping a VM in default subnet"
        az network nsg rule create `
            -g $resourceGroup
            -n allowAAD `
            --nsg-name myNsg `
            --priority 100 `
            --source-address-prefixes 10.0.1.5/32 `
            --destination-address-prefixes AzureActiveDirectory `
            --direction Outbound `
            --protocol * `
            --destination-port-ranges * `
            --description "Allow pod2 to AAD"
        az network nsg rule create `
            -g $resourceGroup
            -n denyInternetFromPod2 `
            --nsg-name myNsg `
            --priority 500 `
            --source-address-prefixes 10.0.1.5/32 `
            --destination-address-prefixes Internet `
            --access deny `
            --protocol * `
            --destination-port-ranges * `
            --description "Deny pod2 to internet"
        az network vnet subnet update `
            -g $resourceGroup `
            -n default `
            --vnet-name $custaVnetName `
            --network-security-group myNsg
        az network vnet subnet update `
            -g $resourceGroup `
            -n $delegatedSubnet `
            --vnet-name $custaVnetName `
            --network-security-group myNsg

        ## Create public IP for NAT Gateway resource
        az network public-ip create `
            --resource-group $resourceGroup `
            --name natGatewayIP `
            --sku standard

        ##  Create NAT Gateway resource
        az network nat gateway create `
            --resource-group $resourceGroup `
            --name natgateway `
            --public-ip-addresses natGatewayIP

        ## Create Cust Vnet (exclusive) for NAT gateway
        az network vnet create `
            --resource-group $resourceGroup `
            --location $region `
            --name $custVnetWithMNAT `
            --address-prefix 10.0.0.0/16 `
            --subnet-name $delegatedSubnet `
            --subnet-prefix 10.0.1.0/24

        ## Attach the subnet from NAT Gateway VNET to NAT gateway resource
        az network vnet subnet update `
            --resource-group $resourceGroup `
            --vnet-name $custVnetWithMNAT  `
            --name  $delegatedSubnet `
            --nat-gateway natgateway

        ## Create Cust Vnet (exclusive) for LB
        az network vnet create `
            --resource-group $resourceGroup `
            --location $region `
            --name $custVnetWithLB `
            --address-prefix 192.168.0.0/16 `
            --subnet-name $delegatedSubnet `
            --subnet-prefix 192.168.1.0/24

        ## Create public IP for LB resource
        az network public-ip create `
            --resource-group $resourceGroup `
            --name swiftlbfrontendip `
            --sku standard

       ## Create LB resource
        az network lb create `
            --resource-group $resourceGroup `
            --name swiftlbforrunner `
            --sku Standard `
            --frontend-ip-name testfrontendip ` 
            --public-ip-address swiftlbfrontendip

       ## Create LB address pool
        az network lb address-pool create `
            --resource-group $resourceGroup `
            --lb-name swiftlbforrunner `
            --name testaddresspool `
            --vnet $custVnetWithLB `
            --backend-address name=addr1 ip-address=192.168.1.4

        az network lb probe create `
            --resource-group $resourceGroup `
            --lb-name swiftlbforrunner `
            --name testProbe `
            --protocol tcp `
            --port 9095
 
        az network lb rule create `
            --resource-group $resourceGroup `
            --lb-name swiftlbforrunner `
            --name testlbrule `
            --protocol Tcp `
            --frontend-ip-name testfrontendip `
            --frontend-port 9095 `
            --backend-pool-name testaddresspool `
            --backend-port 9095 `
            --probe-name testprobe

        ## Replace Cust Vnet with MNAT VNET id in runner-conf.yaml
        $natGwIp = (az network public-ip show -g $resourceGroup -n natGatewayIP | jq.exe -r '.ipAddress')
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_MNAT_PUBLIC_IP", $natGwIp | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        $natVnetId = (az network vnet show -g $resourceGroup -n $custVnetWithMNAT | jq.exe -r '.resourceGuid')
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_CUST_VNET_WITH_MNAT_VNETID", $natVnetId | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## Replace Cust Vnet with LB VNET id in runner-conf.yaml
        $swiftLbIp = (az network public-ip show -g $resourceGroup -n swiftlbfrontendip | jq.exe -r '.ipAddress')
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_SWIFTLB_PUBLIC_IP", $swiftLbIp | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        $custVnetWithLBId = (az network vnet show -g $resourceGroup -n $custVnetWithLB | jq.exe -r '.resourceGuid')
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_CUST_VNET_WITH_SWIFTLB_VNETID", $custVnetWithLBId | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## SWITCH OVER TO SAW/JIT ACCESS -> swiftify and retrieve authToken using $infraVnet, $custVnet, $custVnetWithMNAT and $custVnetWithLB
        Write-Host "Values for delegation...`nInfraVnet: $infraVnet`nCustVnet: $custVnet`ncustVnetWithMNAT: $custVnetWithMNAT`custVnetWithLB: $custVnetWithLB`nDncEP: $k8sEP"

        ## replace the authToken in runner-conf.yaml
        $authToken = "4413692c-e47b-4d62-b137-2c523cc9d87e"
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_AUTHENTICATION_TOKEN", $authToken | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## replace the authToken for cust vnet with MNAT in runner-conf.yaml and continue
        $authToken = "c18d2f22-b1e3-49f7-b462-a5c5bea5372e"
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_AUTHENTICATION_TOKEN_CUST_VNET_WITH_MNAT", $authToken | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        ## replace the authToken for cust vnet with LB in runner-conf.yaml and continue
        $authToken = "74a93c9f-4b49-4293-9e48-7b8c6d9bcd48"
        (Get-Content $configurationFilesPath\runner-conf.yaml) -replace "REPLACE_WITH_AUTHENTICATION_TOKEN_CUST_VNET_WITH_SWIFTLB", $authToken | Out-File $configurationFilesPath\runner-conf.yaml -Encoding ASCII

        Read-Host 'Press any key to continue... (1/2)'
        Read-Host 'Press any key to confirm continuation... (2/2)'

        ## transfer configuration file to master node
        scp $configurationFilesPath\runner-conf.yaml acn@$k8sEP`:/home/acn

        ## take the following output and replace the script value in configure.json IFF we change commands in configure.sh
        ## [run in bash] cat configure-master.sh | gzip -9 | base64 -w 0
        ## update credentials in configure master script

        ## setup CNI in node 1 to retrieve network info from CNS via node IP
        $nodeIP = (az vm list-ip-addresses `
            -g $resourceGroup `
            -n $agentVm1Name | jq.exe -r '.[0].virtualMachine.network.privateIpAddresses[0]')
        (Get-Content $configurationFilesPath\configure-agent.sh) -replace "<NODE-IP>", $nodeIP | Out-File $configurationFilesPath\configure-agent.sh -Encoding ASCII
        az vm extension set `
            -g $resourceGroup `
            --vm-name $agentVm1Name `
            -n customScript `
            --publisher Microsoft.Azure.Extensions `
            --protected-settings $configurationFilesPath\configure-agent.json
        az vm restart `
            -g $resourceGroup `
            -n $agentVm1Name
        ## reset configure-agent.sh before running this second part to configure node 2
        $nodeIP = (az vm list-ip-addresses `
            -g $resourceGroup `
            -n $agentVm2Name | jq.exe -r '.[0].virtualMachine.network.privateIpAddresses[0]')
        (Get-Content $configurationFilesPath\configure-agent.sh) -replace "<NODE-IP>", $nodeIP | Out-File $configurationFilesPath\configure-agent.sh -Encoding ASCII
        az vm extension set `
            -g $resourceGroup `
            --vm-name $agentVm2Name `
            -n customScript `
            --publisher Microsoft.Azure.Extensions `
            --protected-settings $configurationFilesPath\configure-agent.json
        az vm restart `
            -g $resourceGroup `
            -n $agentVm2Name
        Start-Sleep -s 60
        ## set credentials on master to pull private dnc image and apply runner-conf.yaml
        az vm extension set `
            -g $resourceGroup `
            --vm-name $masterVmName `
            -n customScript `
            --publisher Microsoft.Azure.Extensions `
            --protected-settings $configurationFilesPath\configure-master.json

        ## get the external ip for canary client (may take some time before available...)
        Start-Sleep -s 45
        $canaryEP = (az network public-ip list -g $resourceGroup | jq.exe --arg service "$service" '.[] | select(.tags.service==$service) | .name').Replace("`"","")
        az network public-ip update `
            -g $resourceGroup `
            -n $canaryEP `
            --dns-name swiftrunner `
            --allocation-method Static

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