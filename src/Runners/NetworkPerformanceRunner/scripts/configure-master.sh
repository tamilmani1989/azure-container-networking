#!/bin/bash
kubectl apply -f /home/acn/runner-server.yaml
# mv /etc/kubernetes/addons/blobfuse-flexvolume-installer.yaml /home/acn
# mv /etc/kubernetes/addons/coredns.yaml /home/acn
# mv /etc/kubernetes/addons/kube-heapster-deployment.yaml /home/acn
# mv /etc/kubernetes/addons/kubernetes-dashboard-deployment.yaml /home/acn
# mv /etc/kubernetes/addons/kube-metrics-server-deployment.yaml /home/acn
# mv /etc/kubernetes/addons/kube-tiller-deployment.yaml /home/acn
# mv /etc/kubernetes/addons/keyvault-flexvolume-installer.yaml /home/acn
# kubectl delete -f /home/acn/blobfuse-flexvolume-installer.yaml &
# kubectl delete -f /home/acn/keyvault-flexvolume-installer.yaml &
# kubectl delete -f /home/acn/coredns.yaml
# kubectl delete -f /home/acn/kube-heapster-deployment.yaml
# kubectl delete -f /home/acn/kubernetes-dashboard-deployment.yaml
# kubectl delete -f /home/acn/kube-metrics-server-deployment.yaml
# kubectl delete -f /home/acn/kube-tiller-deployment.yaml
sleep 45
