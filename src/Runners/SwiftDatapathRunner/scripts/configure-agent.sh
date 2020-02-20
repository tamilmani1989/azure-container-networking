#!/bin/bash
apt install -y openvswitch-switch
sed -i '/"enableSnatOnHost":true/a "cnsurl":"http://<NODE-IP>:10090",' /etc/cni/net.d/10-azure.conflist
mkdir /var/lib/azure-network