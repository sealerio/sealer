# Sealer -- Build, Share and Run Any Distributed Applications

[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/sealerio/sealer/blob/master/LICENSE)
[![Go](https://github.com/sealerio/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/go.yml)
[![Release](https://github.com/sealerio/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/release.yml)
[![GoDoc](https://godoc.org/github.com/sealerio/sealer?status.svg)](https://godoc.org/github.com/sealerio/sealer)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5205/badge)](https://bestpractices.coreinfrastructure.org/en/projects/5205)
[![Twitter](https://img.shields.io/badge/Follow-sealer-1DA1F2?logo=twitter)](https://twitter.com/sealer_oss)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsealerio%2Fsealer.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsealerio%2Fsealer?ref=badge_shield)
[![codecov](https://codecov.io/gh/sealerio/sealer/branch/main/graph/badge.svg?token=LH8XUR5YPL)](https://codecov.io/gh/sealerio/sealer)

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
wget https://github.com/sealerio/sealer/releases/download/v0.9.0/sealer-v0.9.0-linux-amd64.tar.gz && \
tar zxvf sealer-v0.9.0-linux-amd64.tar.gz && mv sealer /usr/bin
```

Build a ClusterImage with Kubernetes dashboard:

Kubefile:

```shell script
# base ClusterImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this ClusterImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

Build it:

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest .
```

Make it run:

```shell script
# sealer will install a kubernetes on host 192.168.0.2 then apply the dashboard manifests
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest --masters 192.168.0.2 --passwd xxx
# check the pod
kubectl get pod -A|grep dashboard
```

Push the ClusterImage to the registry

```shell script
# you can push the ClusterImage to docker hub, Ali ACR, or Harbor
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

## User guide

Sealer provides a valid image list:

| version |                   clusterimage with CNI(calico)                     |                                   clusterimage                            |
| :-----  | :-------------------------------------------------------------------| :-------------------------------------------------------------------------|
| 0.8.6   | registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15-0.8.6| registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15-0.8.6-alpha|
| main    | registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15      | registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.22.15-alpha      |

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
