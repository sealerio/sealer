# 集群镜像列表

我们已经构建出一些可复用的集群镜像供用户使用，比如数据库，监控，消息队列等

## Overview

我们使用OpenEBS 作为默认存储，提供各种有状态应用动态创建PV.

### 基础工具

* registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0

### 存储

* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-cstor:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-block:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-file:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/ceph-object:v16.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/minio:2021.6.17

### 网络

* registry.cn-qingdao.aliyuncs.com/sealer-apps/ingress-nginx-controller:v1.0.0

### 数据库

* registry.cn-qingdao.aliyuncs.com/sealer-apps/mysql:8.0.26
* registry.cn-qingdao.aliyuncs.com/sealer-apps/redis:6.2.5
* registry.cn-qingdao.aliyuncs.com/sealer-apps/mongodb:4.4.8
* registry.cn-qingdao.aliyuncs.com/sealer-apps/postgresql:11.12.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cassandra:4.0.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/tidb:v1.2.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/cockroach:v21.1.7

### 消息队列

* registry.cn-qingdao.aliyuncs.com/sealer-apps/kafka:2.8.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/zookeeper:3.7.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/rocketmq:4.5.0

### 其它镜像

* registry.cn-qingdao.aliyuncs.com/sealer-apps/dashboard:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/prometheus-stack:v2.28.1
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-promtail:v2.2.0
* registry.cn-qingdao.aliyuncs.com/sealer-apps/loki-stack-fluentbit:v2.2.0

## 如何使用

### 创建集群

可以直接修改以下Clusterfile中的image字段，然后使用 `sealer apply -f Clusterfile` 去启动集群，以prometheus为例：

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

如果你需要持久化存储，我们提供openebs块存储，但是所有节点需要安装 iSCSI client,好在sealer提供的插件能力可以支持在每个节点执行一些指定操作，以下以在centos上
安装iSCSI client为例，只需要在Clusterfile中添加如下插件配置：

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

具体每个CloudImage本身的访问方式与使用方式请参考对应的[readme文件](https://github.com/alibaba/sealer/tree/main/applications)