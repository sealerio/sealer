# Sealer -- Build, Share and Run Any Distributed Applications

[![License](https://img.shields.io/badge/license-Apache%202-brightgreen.svg)](https://github.com/sealerio/sealer/blob/master/LICENSE)
[![Go](https://github.com/sealerio/sealer/actions/workflows/go.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/go.yml)
[![Release](https://github.com/sealerio/sealer/actions/workflows/release.yml/badge.svg)](https://github.com/sealerio/sealer/actions/workflows/release.yml)
[![GoDoc](https://godoc.org/github.com/sealerio/sealer?status.svg)](https://godoc.org/github.com/sealerio/sealer)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5205/badge)](https://bestpractices.coreinfrastructure.org/en/projects/5205)
[![Twitter](https://img.shields.io/badge/Follow-sealer-1DA1F2?logo=twitter)](https://twitter.com/sealer_oss)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsealerio%2Fsealer.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsealerio%2Fsealer?ref=badge_shield)

[简体中文](./docs/README_zh.md)

## Contents

* [Introduction](#introduction)
* [Quick Start](#quick-start)
* [Contributing](./CONTRIBUTING.md)
* [FAQ](./FAQ.md)
* [Adopters](./Adopters.md)
* [LICENSE](LICENSE)
* [Code of conduct](./code-of-conduct.md)

## Introduction

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes.

![image](https://user-images.githubusercontent.com/8912557/117263291-b88b8700-ae84-11eb-8b46-838292e85c5c.png)

> Concept

* ClusterImage : like Dockerimage, but the rootfs is kubernetes, and contains all the dependencies(docker images,yaml files or helm chart...) your application needs.
* Kubefile : the file describe how to build a ClusterImage.
* Clusterfile : the config of using ClusterImage to run a cluster.

![image](https://user-images.githubusercontent.com/8912557/117400612-97cf3a00-af35-11eb-90b9-f5dc8e8117b5.png)

We can write a Kubefile, and build a ClusterImage, then using a Clusterfile to run a cluster.

sealer[ˈsiːlər] provides the way for distributed application package and delivery based on kubernetes.

It solves the delivery problem of complex applications by packaging distributed applications and dependencies(like database,middleware) together.

For example, build a dashboard ClusterImage:

Kubefile:

```shell script
# base ClusterImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this ClusterImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

Build dashobard ClusterImage:

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

Push the ClusterImage to the registry

```shell script
# you can push the ClusterImage to docker hub, Ali ACR, or Harbor
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

## Usage scenarios & features

* [x] An extremely simple way to install kubernetes and other software in the kubernetes ecosystem in a production or offline environment.
* [x] Through Kubefile, you can easily customize the kubernetes ClusterImage to package the cluster and applications, and submit them to the registry.
* [x] Powerful life cycle management capabilities, to perform operations such as cluster upgrade, cluster backup and recovery, node expansion and contraction in unimaginable simple ways
* [x] Very fast, complete cluster installation within 3 minutes
* [x] Support ARM x86, v1.20 and above versions support containerd, almost compatible with all Linux operating systems that support systemd
* [x] Does not rely on ansible haproxy keepalived, high availability is achieved through ipvs, takes up less resources, is stable and reliable
* [x] Many ecological software images can be used directly, like prometheus mysql..., and you can combine then together.

## Quick start

Install a kubernetes cluster

```shell script
#install Sealer binaries
wget https://github.com/sealerio/sealer/releases/download/v0.8.5/sealer-v0.8.5-linux-amd64.tar.gz && \
tar zxvf sealer-v0.8.5-linux-amd64.tar.gz && mv sealer /usr/bin
#run a kubernetes cluster
sealer run kubernetes:v1.19.8 --masters 192.168.0.2 --passwd xxx
```

## User guide

[get started](http://sealer.cool/docs/getting-started/introduction.html)

## Official website

[official website](http://sealer.cool)

## Developing Sealer

* [contributing guide](./CONTRIBUTING.md)
* [贡献文档](./docs/contributing_zh.md)

## Communication Channels

* CNCF Mailing List: to be added.
* Twitter: [@sealer](https://twitter.com/sealer_oss)
* DingTalk Group Number: 34619594

<!-- markdownlint-disable -->
<div align="center">
  <img src="https://user-images.githubusercontent.com/9465626/167836908-65ca12e0-4600-44d4-b3bd-8b93d27600fb.png" width="300" title="dingtalk">
</div>
<!-- markdownlint-restore -->

## Code of Conduct

sealer follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## License

Sealer is licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for the full license text.

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fsealerio%2Fsealer.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fsealerio%2Fsealer?ref=badge_large)