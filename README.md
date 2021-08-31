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
wget https://github.com/alibaba/sealer/releases/download/v0.3.4/sealer-v0.3.4-linux-amd64.tar.gz && \
tar zxvf sealer-v0.3.4-linux-amd64.tar.gz && mv sealer /usr/bin
#run a kubernetes cluster 
sealer run kubernetes:v1.19.9 --masters 192.168.0.2 --passwd xxx
```

# User guide

[get started](docs/user-guide/get-started.md)

# Developing Sealer

* [contributing guide](./CONTRIBUTING.md)
* [贡献文档](./docs/contributing_zh.md)

# Maintainers&Partners

<img src="https://img.alicdn.com/tfs/TB13DzOjXP7gK0jSZFjXXc5aXXa-212-48.png" width="100px" />
<img src="https://cdn.zcygov.cn/logo.png" width="100px" />
<img src="http://harmonycloud.cn/uploads/images/202105/338aa0549c307208539755b8d2e0d352.png" width="100px" />
