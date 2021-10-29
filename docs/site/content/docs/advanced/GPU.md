+++
title = "GPU supported"
description = "Using GPU with nvidia-plugin CloudImage"
date = 2021-05-01T19:30:00+00:00
updated = 2021-05-01T19:30:00+00:00
draft = false
weight = 30
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Using GPU with nvidia-plugin CloudImage"
toc = true
top = false
+++

# GPU with nvidia-plugin

## Preparation

1. install nvidia driver on your host.
2. install the latest version of sealer on your host.

## How to build it

we provide GPU base image in our official registry
named `registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8`.you can use is directly. meanwhile, we
provide the build context in the applications' directory. it can be adjusted it per your request.

run below command to rebuild it.

`sealer build -f Kubefile -t registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8 -b lite .`

## How to apply it

1. Modify the Clusterfile according to your infra environment,here is the Clusterfile for example.

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes-nvidia:v1.19.8
  # if Using local host to apply, the provider should be BAREMETAL
  provider: BAREMETAL
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd: *
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: ""
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: ""
    # ssh login user
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  masters:
    ipList:
      - 172.22.82.184
```

2. run command `sealer apply -f Clusterfile` to apply the GPU cluster. it will take few minutes.

## How to check the result

1. check the pod status to run `kubectl get pods -n kube-system nvidia-device-plugin`, you can find the pod in Running
   status.
2. get the node details to run `kubectl describe node`, if `nvidia.com/gpu` shows on 'Allocated resources' section,you
   get a k8s cluster with GPU. 
