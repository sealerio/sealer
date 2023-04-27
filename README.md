# Sealer -- Build, Share and Run Any Distributed Applications

[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/sealerio/sealer/blob/master/LICENSE)
[![Go](https://github.com/sealerio/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/go.yml)
[![Release](https://github.com/sealerio/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/release.yml)
[![GoDoc](https://godoc.org/github.com/sealerio/sealer?status.svg)](https://godoc.org/github.com/sealerio/sealer)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5205/badge)](https://bestpractices.coreinfrastructure.org/en/projects/5205)
[![Twitter](https://img.shields.io/badge/Follow-sealer-1DA1F2?logo=twitter)](https://twitter.com/sealer_oss)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsealerio%2Fsealer.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsealerio%2Fsealer?ref=badge_shield)
[![codecov](https://codecov.io/gh/sealerio/sealer/branch/main/graph/badge.svg?token=LH8XUR5YPL)](https://codecov.io/gh/sealerio/sealer)
[![](https://img.shields.io/badge/Sealer-Check%20Your%20Contribution-orange)](https://opensource.alibaba.com/contribution_leaderboard/details?projectValue=sealer)

## Contents

* [Introduction](#introduction)
* [Quick Start](#quick-start)
* [Contributing](./CONTRIBUTING.md)
* [FAQ](./FAQ.md)
* [Adopters](./Adopters.md)
* [LICENSE](LICENSE)
* [Code of conduct](./code-of-conduct.md)

## Introduction

Sealer[ˈsiːlər] provides a new way of distributed application delivery which is reducing the difficulty and complexity by packaging Kubernetes cluster and all application's dependencies into one ClusterImage.

We can write a Kubefile to build the ClusterImage, and use it to deliver your applications with embedded Kubernetes through Clusterfile.

![image](https://user-images.githubusercontent.com/8912557/117263291-b88b8700-ae84-11eb-8b46-838292e85c5c.png)

> Concept

* Kubefile: a file that describes how to build a ClusterImage.
* ClusterImage: like docker image, and it contains all the dependencies(container images,yaml files or helm chart...) of your application needed.
* Clusterfile: a file that describes how to run a ClusterImage.

![image](https://user-images.githubusercontent.com/8912557/117400612-97cf3a00-af35-11eb-90b9-f5dc8e8117b5.png)

## Awesome features

* [x] Simplicity: Packing the distributed application into ClusterImage with few instructions.
* [x] Efficiency: Launching the k8s-based application through ClusterImage in minutes.
* [x] Scalability: Powerful cluster and image life cycle management, such as cluster scale, upgrade, image load, save and so on.
* [x] Compatibility: Multi-arch delivery Supporting. Such as AMD, ARM with common Linux distributions.
* [x] Iterative: Incremental operations on ClusterImage is like what container image behaves.

## Quick start

Download sealer binary file.

```shell script
#install Sealer binaries
wget https://github.com/sealerio/sealer/releases/download/v0.9.3/sealer-v0.9.3-linux-amd64.tar.gz && \
tar zxvf sealer-v0.9.3-linux-amd64.tar.gz && mv sealer /usr/bin
```

## Install a kubernetes cluster

```shell
# run a kubernetes cluster
sealer run docker.io/sealerio/kubernetes:v1-22-15-sealerio-2 \
  --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
  --nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
```

## Build an sealer image

Kubefile:

```shell
FROM docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
APP mysql https://charts/mysql.tgz
APP elasticsearch https://charts/elasticsearch.tgz
APP redis local://redis.yaml
APP businessApp local://install.sh
LAUNCH ["calico", "mysql", "elasticsearch", "redis", "businessApp"]
```

or

```shell
FROM docker.io/sealerio/kubernetes:v1-22-15-sealerio-2
COPY mysql.tgz .
COPY elasticsearch.tgz .
COPY redis.yaml .
COPY install.sh .
CMDS ["sh application/apps/calico/calico.sh", "helm install mysql.tgz", "helm install elasticsearch.tgz", "kubectl apply -f redis.yaml", "bash install.sh"]
```

build command:

> NOTE: --type=kube-installer is the default value for sealer build

```shell
sealer build -f Kubefile -t my-kubernetes:1.0.0 .
```

## Build an app image

nginx.yaml:

```shell
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-nginx
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      run: my-nginx
  template:
    metadata:
      labels:
        run: my-nginx
    spec:
      containers:
        - name: my-nginx
          image: nginx
          ports:
            - containerPort: 80
```

Kubefile:

```shell
FROM scratch
APP nginx local://nginx.yaml
LAUNCH ["nginx"]
```

```shell
sealer build -f Kubefile -t sealer-io/nginx:latest --type app-installer
```

## Run the app image

```shell
sealer run sealer-io/nginx:latest
# check the pod
kubectl get pod -A
```

## Push the app image to the registry

```shell
# you can push the app image to docker hub, Ali ACR, or Harbor
sealer tag sealer-io/nginx:latest {registryDomain}/sealer-io/nginx:latest
sealer push {registryDomain}/sealer-io/nginx:latest
```

## Clean the cluster

Some information of the basic settings will be written to the Clusterfile and stored in /root/.sealer/Clusterfile.

```shell
sealer delete -a
```

## User guide

Sealer provides a valid image list:

| version  |                              image                                  |                  Arch                   |                                                           OS                                                        |              Network plugins            |             container runtime           |
| :------: | :-----------------------------------------------------------------: | :-------------------------------------: | :-----------------------------------------------------------------------------------------------------------------: | :-------------------------------------: | :-------------------------------------: |
| v0.8.6   | registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15-0.8.6|                   x86                   |     CentOS/RHEL 7.5<br>CentOS/RHEL 7.6<br>CentOS/RHEL 7.7<br>CentOS/RHEL 7.8<br>CentOS/RHEL 7.9<br>Ubuntu 20.04     |                 calico                  |            hack docker v19.03.14        |
| v0.9.3   | docker.io/sealerio/kubernetes:v1-18-3-sealerio-2                    |                x86/arm64                |     CentOS/RHEL 7.5<br>CentOS/RHEL 7.6<br>CentOS/RHEL 7.7<br>CentOS/RHEL 7.8<br>CentOS/RHEL 7.9<br>Ubuntu 20.04     |                 calico                  |            hack docker v19.03.14        |
| v0.9.3   | docker.io/sealerio/kubernetes:v1-20-4-sealerio-2                    |                x86/arm64                |     CentOS/RHEL 7.5<br>CentOS/RHEL 7.6<br>CentOS/RHEL 7.7<br>CentOS/RHEL 7.8<br>CentOS/RHEL 7.9<br>Ubuntu 20.04     |                 calico                  |            hack docker v19.03.14        |
| v0.9.3   | docker.io/sealerio/kubernetes:v1-22-15-sealerio-2                   |                x86/arm64                |     CentOS/RHEL 7.5<br>CentOS/RHEL 7.6<br>CentOS/RHEL 7.7<br>CentOS/RHEL 7.8<br>CentOS/RHEL 7.9<br>Ubuntu 20.04     |                 calico                  |            hack docker v19.03.14        |

[get started](http://sealer.cool/docs/getting-started/introduction.html)

## Official website

[official website](http://sealer.cool)

## Developing Sealer

* [contributing guide](./CONTRIBUTING.md)

## Communication Channels

* CNCF Mailing List: to be added.
* Twitter: [@sealer](https://twitter.com/sealer_oss)
* DingTalk Group Number: 34619594

<!-- markdownlint-disable -->
<div align="center">
  <img src="https://user-images.githubusercontent.com/31209634/199941518-82f88ba5-d13c-420c-9197-95a422f6b543.JPG" width="300" title="dingtalk">
</div>
<!-- markdownlint-restore -->

## Code of Conduct

sealer follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## License

Sealer is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsealerio%2Fsealer.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsealerio%2Fsealer?ref=badge_large)
