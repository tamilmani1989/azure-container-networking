#!/bin/bash
kubectl create secret docker-registry msint.azurecr.io-key --docker-server=msint.azurecr.io --docker-username=acb4af04-305f-4b23-90e7-0e31b7a3f02f --docker-password=REPLACEPASSWORD --docker-email=acn@microsoft.com
kubectl apply -f /home/acn/runner-conf.yaml
sleep 45
POD=$(kubectl get pod -l app=swift-runner -o jsonpath="{.items[0].metadata.name}")
kubectl cp /home/acn/storage.json $POD:/
kubectl cp /home/acn/subnet.json $POD:/
