# GPU CloudImage

## Preparation

1. Install nvidia driver on your host.
2. Install the latest version of sealer on your host.

## How to build it

We provide GPU base image in our official registry
named `registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8`.you can use is directly. meanwhile, we
provide the build context in the applications' directory. it can be adjusted it per your request.

Run below command to rebuild it.

`sealer build -f Kubefile -t registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8 -m lite .`

## How to apply it

1. Modify the Clusterfile according to your infra environment,here is the Clusterfile for example.

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8
  ssh:
    passwd: xxx
  hosts:
    - ips: [ 192.168.0.2,192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
```

2. Run command `sealer apply -f Clusterfile` to apply the GPU cluster. it will take few minutes.

## How to check the result

1. Check the pod status to run `kubectl get pods -n kube-system nvidia-device-plugin`, you can find the pod in Running
   status.
2. Get the node details to run `kubectl describe node`, if `nvidia.com/gpu` shows on 'Allocated resources' section,you
   get a k8s cluster with GPU.
