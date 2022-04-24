# How to using nvdia GPU Operator manually

## Install GPU driver on your host

## Add helm repository

```
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 \
   && chmod 700 get_helm.sh \
   && ./get_helm.sh

helm repo add nvidia https://helm.ngc.nvidia.com/nvidia \
   && helm repo update
```

## Install Bare-metal/Passthrough on Ubuntu

```
helm install --wait --generate-name \
     -n gpu-operator --create-namespace \
     nvidia/gpu-operator --set driver.enabled=false
```

## Or Bare-metal/Passthrough with default configurations on CentOS

```
helm install --wait --generate-name \
     -n gpu-operator --create-namespace \
     nvidia/gpu-operator \
     --set toolkit.version=1.7.1-centos7 --set driver.enabled=false
```

## Check & run sample

```
[root@iZbp18um4r5nm2pu8g9yvwZ ~]# kubectl get pod -n gpu-operator
NAME                                                              READY   STATUS      RESTARTS   AGE
gpu-feature-discovery-tmgs5                                       1/1     Running     0          21m
gpu-operator-1650854972-node-feature-discovery-master-644b2j9d9   1/1     Running     0          21m
gpu-operator-1650854972-node-feature-discovery-worker-6jphc       1/1     Running     0          21m
gpu-operator-794b8c8ddc-dswrq                                     1/1     Running     0          21m
nvidia-container-toolkit-daemonset-rlfct                          1/1     Running     0          21m
nvidia-cuda-validator-7cfgq                                       0/1     Completed   0          19m
nvidia-dcgm-exporter-j45bz                                        1/1     Running     0          21m
nvidia-device-plugin-daemonset-pglrm                              1/1     Running     0          21m
nvidia-device-plugin-validator-c7vsx                              0/1     Completed   0          18m
nvidia-operator-validator-9kn52                                   1/1     Running     0          21m
```

Check node resource:

```
kubectl describe node xxx
...
Capacity:
  cpu:                4
  ephemeral-storage:  41152812Ki
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             15233940Ki
  nvidia.com/gpu:     1
  pods:               110
Allocatable:
  cpu:                4
  ephemeral-storage:  37926431477
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             15131540Ki
  nvidia.com/gpu:     1
  pods:               110
...
```

Run a sample pod:

```
cat << EOF | kubectl create -f -
apiVersion: v1
kind: Pod
metadata:
  name: cuda-vectoradd
spec:
  restartPolicy: OnFailure
  containers:
  - name: cuda-vectoradd
    image: "nvidia/samples:vectoradd-cuda11.2.1"
    resources:
      limits:
         nvidia.com/gpu: 1
EOF
```

```
[root@iZbp18um4r5nm2pu8g9yvwZ ~]# kubectl logs cuda-vectoradd
[Vector addition of 50000 elements]
Copy input data from the host memory to the CUDA device
CUDA kernel launch with 196 blocks of 256 threads
Copy output data from the CUDA device to the host memory
Test PASSED
Done
```
