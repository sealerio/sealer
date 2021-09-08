# Prerequisites

* install sealer in your machine
* if your want to run cloud image on alibaba cloud, need AK,SK first.

## Overview

We choose OpenEBS Jiva or OpenEBS LocalPV as default persistence storage to enable Stateful applications to easily access Dynamic Local PVs
or Replicated PVs. More details about the application can be found in its manifest directory.

### Cloud image list

#### Install tools image

* registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0

#### Infra image

* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-cstor:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ingress-nginx-controller:v1.0.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-block:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-file:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-object:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/minio:2021.6.17

#### Database image

* registry.cn-qingdao.aliyuncs.com/sealer-apps/mysql:8.0.26
* registry.cn-qingdao.aliyuncs.com/sealer-apps/redis:6.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/mongodb:4.4.8
* registry.cn-qingdao.aliyuncs.com/sealer-apps/postgresql:11.12.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cassandra:4.0.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/tidb:v1.2.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cockroach:v21.1.7

#### Message queue image

* registry.cn-qingdao.aliyuncs.com/sealer-apps/kafka:2.8.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/zookeeper:3.7.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/rocketmq:4.5.0

#### Application image

* registry.cn-qingdao.aliyuncs.com/sealer-apps/dashboard:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/prometheus-stack:v2.28.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-promtail:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-fluentbit:v2.2.0



## How to run it

### Apply a cluster

you can modify the image name and save it as "clusterfile.yaml", then run sealer apply
cmd  `sealer apply -f clusterfile.yaml`

```yaml
apiVersion: zlink.aliyun.com/v1alpha1
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  image: { your cloud image name }
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
      - "100"
    memory: "4"
    systemDisk: "100"
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
      - "100"
    memory: "4"
    systemDisk: "100"
  provider: ALI_CLOUD
  ssh:
    passwd: Seadent123
    pk: xxx
    pkPasswd: xxx
    user: root
```

if you want to apply a cloud image which need persistence storage. we provide openebs as cloud storage backend. OpenEBS
provides block volume support through the iSCSI protocol. Therefore, the iSCSI client (initiator) presence on all
Kubernetes nodes is required. Choose the platform below to find the steps to verify if the iSCSI client is installed and
running or to find the steps to install the iSCSI client.For openebs, different storage engine need to config different
prerequisite. more to see [openebs website](https://docs.openebs.io/).

We provide plugin mechanism, you only need to append below example to "clusterfile.yaml" and apply them together.

For example, if we use jiva engine as storage backend :

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=node
  data: |
    if type yum >/dev/null 2>&1;then
    yum -y install iscsi-initiator-utils
    systemctl enable iscsid
    systemctl start iscsid
    elif type apt-get >/dev/null 2>&1;then
    apt-get update
    apt-get -y install open-iscsi
    systemctl enable iscsid
    systemctl start iscsid
    fi
---
```

## How to use it

See README.md of each application for more details.

## How to rebuild it

Use it as base image to build another useful image .For example, use helm image to build a mysql CloudImage:

### Use helm

Kubefile:

```shell
# base CloudImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0
# add helm repo and run helm install
CMD helm repo add bitnami https://charts.bitnami.com/bitnami && helm install my-mysql bitnami/mysql --version 8.8.5
```

run below command to build a mysql cloud image

```shell
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-apps/mysql:8.8.5 -b cloud .
```

### Use manifest

Kubefile:

See each manifest yaml file under application manifest directory for details , and modify it according to your needs.

Then run below command to rebuild it

```shell
sealer build -t {Your Image Name} -f Kubefile -b cloud .
```
