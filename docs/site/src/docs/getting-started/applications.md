# Applications CloudImage

Before using the sealer official applications, you need to install the [sealer](https://github.com/sealerio/sealer).

## Overview

We choose OpenEBS Jiva or OpenEBS LocalPV as default persistence storage to enable Stateful applications to easily access Dynamic Local PVs
or Replicated PVs. More details about the application can be found in its manifest directory.

### Cloud image list

#### Toolkit images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0

#### Storage images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-cstor:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-block:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-file:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-object:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/minio:2021.6.17

#### Network images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/ingress-nginx-controller:v1.0.0

#### Database images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/mysql:8.0.26
* registry.cn-qingdao.aliyuncs.com/sealer-apps/redis:6.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/mongodb:4.4.8
* registry.cn-qingdao.aliyuncs.com/sealer-apps/postgresql:11.12.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cassandra:4.0.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/tidb:v1.2.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cockroach:v21.1.7

#### Message queue images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/kafka:2.8.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/zookeeper:3.7.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/rocketmq:4.5.0

#### Other images

* registry.cn-qingdao.aliyuncs.com/sealer-apps/dashboard:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/prometheus-stack:v2.28.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-promtail:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-fluentbit:v2.2.0

## How to run it

### Apply a cluster

you can modify the image name and save it as "Clusterfile", then run sealer apply
cmd  `sealer apply -f Clusterfile`, for example install prometheus stack:

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-apps/prometheus-stack:v2.28.1
  ssh:
    passwd: xxx
  hosts:
    - ips: [ 192.168.0.2,192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
```

if you want to apply a cloud image which need persistence storage. we provide openebs as cloud storage backend. OpenEBS
provides block volume support through the iSCSI protocol. Therefore, the iSCSI client (initiator) presence on all
Kubernetes nodes is required. Choose the platform below to find the steps to verify if the iSCSI client is installed and
running or to find the steps to install the iSCSI client.For openebs, different storage engine need to config different
prerequisite. more to see [openebs website](https://openebs.io/).

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