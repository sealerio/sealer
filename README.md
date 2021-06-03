[简体中文](./docs/README_zh.md)

[![Go](https://github.com/alibaba/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/go.yml)
[![Release](https://github.com/alibaba/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/alibaba/sealer/actions/workflows/release.yml)

# What is sealer

**Build distributed application, share to anyone and run anywhere!!!**

![image](https://user-images.githubusercontent.com/8912557/117263291-b88b8700-ae84-11eb-8b46-838292e85c5c.png)

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes. 

It solves the delivery problem of complex applications by packaging distributed applications and dependencies(like database,middleware) together.

> Concept

* CloudImage : like Dockerimage, but the rootfs is kubernetes, and contains all the dependencies(docker images,yaml files or helm chart...) your application needs.
* Kubefile : the file describe how to build a CloudImage.
* Clusterfile : the config of using CloudImage to run a cluster.

![image](https://user-images.githubusercontent.com/8912557/117400612-97cf3a00-af35-11eb-90b9-f5dc8e8117b5.png)


We can write a Kubefile, and build a CloudImage, then using a Clusterfile to run a cluster.

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes. 

It solves the delivery problem of complex applications by packaging distributed applications and dependencies(like database,middleware) together.

For example, build a dashboard CloudImage:

Kubefile:

```shell script
# base CloudImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this CloudImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

Build dashobard CloudImage:

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest .
```

Run a kubernetes cluster with dashboard:

```shell script
# sealer will install a kubernetes on host 192.168.0.2 then apply the dashboard manifests
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest --masters 192.168.0.2 --passwd xxx
# check the pod
kubectl get pod -A|grep dashboard
```

Push the CloudImage to the registry

```shell script
# you can push the CloudImage to docker hub, Ali ACR, or Harbor
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
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
#install Sealer binaries
wget https://github.com/alibaba/sealer/releases/download/v0.1.4/sealer-0.1.4-linux-amd64.tar.gz && \
tar zxvf sealer-0.1.4-linux-amd64.tar.gz && mv sealer /usr/bin
#run a kubernetes cluster 
sealer run kubernetes:v1.19.9 --masters 192.168.0.2 --passwd xxx
```

Install a cluster on public cloud(now support alicloud):

```shell script
export ACCESSKEYID=xxx
export ACCESSKEYSECRET=xxx
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

Or specify the number of nodes to run the cluster

```shell script
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest \
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
sealer inspect -c registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

Use Clusterfile to set up a k8s cluster

## Scenario 1. Install on an existing server, the provider type is BAREMETAL

Clusterfile content:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: BAREMETAL
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd:
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: xxx
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    ipList:
     - 172.20.125.234
     - 172.20.126.5
     - 172.20.126.6
  nodes:
    ipList:
     - 172.20.126.8
     - 172.20.126.9
     - 172.20.126.10
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
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: ALI_CLOUD
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd:
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: xxx
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
```

## clean the cluster

Some information of the basic settings will be written to the Clusterfile and stored in /root/.sealer/[cluster-name]/Clusterfile.

```shell script
sealer delete -f /root/.sealer/my-cluster/Clusterfile
```

# Developing Sealer

* [contributing guide](./CONTRIBUTING.md)
* [贡献文档](./docs/contributing_zh.md)
