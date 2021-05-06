[简体中文](./docs/README_zh.md)

[![Go](https://github.com/alibaba/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/go.yml)
[![Release](https://github.com/alibaba/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/release.yml)

# What is CloudImage 

![image](https://user-images.githubusercontent.com/8912557/117261089-5467c380-ae82-11eb-8dd8-1163c1a74b10.png)

Docker can build a rootfs+application of an operating system into a container image. 
Sealer regards kubernetes as an operating system. 
The image created on this higher abstraction is a CloudImage. 
Realize the Build share run of the entire cluster!!!

![image](https://user-images.githubusercontent.com/8912557/117263291-b88b8700-ae84-11eb-8b46-838292e85c5c.png)

With CloudImage, it will be extremely simple for users to practice cloud native ecological technology, such as:

> Install a kubernetes cluster:

```shell script
sealer run kubernetes:1.19.2 # Run a kubernetes cluster on the public cloud
sealer run kubernetes:1.19.2 --master 3 --node 3 # Run a kuberentes cluster with a specified number of nodes on the public cloud
```

> Install to an existing machine

```shell script
sealer run kuberntes:1.19.2 --master 192.168.0.2,192.168.0.3,192.168.0.4 --node 192.168.0.5,192.168.0.6,192.168.0.7
```

> Install prometheus cluster

```shell script
sealer run prometheus:2.26.0
```

The above command can help you install a kubernetes cluster that includes prometheus. 
Similarly, other software such as istio ingress grafana can be run in this way.

It's not over yet, the best thing about Sealer is that it is very convenient for users to customize a CloudImage, 
which is described and built through a file like Dockerfile:

Kubefile:

```shell script
FROM registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.6
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer/dashboard:latest.
```

Then a CloudImage containing the dashboard is created, which can be run or shared with others.

Push the created CloudImage to the registry, which is compatible with the docker image registry, 
so you can push the CloudImage to docker hub, Ali ACR, or Harbor

```shell script
sealer push registry.cn-qingdao.aliyuncs.com/sealer/dashboard:latest
```

# Usage scenarios & features

- [x] An extremely simple way to install kubernetes and other software in the kubernetes ecosystem in a production or offline environment. 
- [x] Through Kubefile, you can easily customize the kubernetes CloudImage to package the cluster and applications, and submit them to the registry.  
- [x] Powerful life cycle management capabilities, to perform operations such as cluster upgrade, cluster backup and recovery, node expansion and contraction in unimaginable simple ways 
- [x] Very fast, complete cluster installation within 3 minutes 
- [x] Support ARM x86, v1.20 and above versions support containerd, almost compatible with all Linux operating systems that support systemd 
- [x] Does not rely on ansible haproxy keepalived, high availability is achieved through ipvs, takes up less resources, is stable and reliable 
- [x] There are very few in the official warehouse. Many ecological software images can be used directly, including all dependencies, one-click installation

# Quick start

Install a kubernetes cluster

```shell script
sealer run kubernetes:v1.19.2 --master 192.168.0.2
```

If it is installed on the cloud:

```shell script
export ACCESSKEYID=xxx
export ACCESSKEYSECRET=xxx
sealer run registry.cn-qingdao.aliyuncs.com/sealer/dashboard:latest
```

Or specify the number of nodes to run the cluster

```shell script
sealer run registry.cn-qingdao.aliyuncs.com/sealer/dashboard:latest \
  --masters 3 --nodes 3
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                    STATUS ROLES AGE VERSION
izm5e42unzb79kod55hehvz Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r7z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r8z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r9z Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7raz Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7rbz Ready <none> 18h v1.16.9
```

View the default startup configuration of the CloudImage:

```shell script
sealer config registry.cn-qingdao.aliyuncs.com/sealer/dashboard:latest
```

Use Clusterfile to pull up a k8s cluster
Use the provided official basic image (sealer/cloudrootfs:v1.16.9-alpha.6) to quickly pull up a k8s cluster.

## Scenario 1. Install on an existing server, the provider type is BAREMETAL

Clusterfile content:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5
  provider: BAREMETAL
  ssh:
    passwd:
    pk: xxx
    pkPasswd: xxx
    user: root
  network:
    interface: eth0
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    -aliyun-inc.com
    -10.0.0.2
    
  masters:
    ipList:
     -172.20.125.234
     -172.20.126.5
     -172.20.126.6
  nodes:
    ipList:
     -172.20.126.8
     -172.20.126.9
     -172.20.126.10
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# sealer apply -f Clusterfile
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
```

## Scenario 2. Automatically apply for Alibaba Cloud server for installation, provider: ALI_CLOUD Clusterfile:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer/cloudrootfs:v1.16.9-alpha.5
  provider: ALI_CLOUD
  ssh:
    passwd:
    pk: xxx
    pkPasswd: xxx
    user: root
  network:
    interface: eth0
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    -aliyun-inc.com
    -10.0.0.2
    
  masters:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    -100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    -100
```

## Release the cluster

Some source information of the basic settings will be written to the Clusterfile and stored in /root/.sealer/[cluster-name]/Clusterfile, so the cluster can be released as follows:

```shell script
sealer delete -f /root/.sealer/my-cluster/Clusterfile
```
