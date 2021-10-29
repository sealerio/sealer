#!/bin/bash
kubectl taint nodes --all node-role.kubernetes.io/master-
# wait taint ready
sleep 45
kubectl apply -f ./manifests/nvidia-device-plugin.yml
