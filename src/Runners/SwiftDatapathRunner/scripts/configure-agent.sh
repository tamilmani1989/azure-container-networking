#!/bin/bash
apt install -y openvswitch-switch
wget -L https://raw.githubusercontent.com/Azure/azure-container-networking/master/cni/azure-linux-multitenancy.conflist -P /home/acn/