# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* Three StatefulSet replicas for zookeeper cluster which requests "10Gi" storage.

## How to use it

ZooKeeper can be accessed via port 2181 on the following DNS name from within your cluster:

```
my-zookeeper.zookeeper-system.svc.cluster.local
```

To connect to your ZooKeeper server run the following commands:

```
export POD_NAME=$(kubectl get pods --namespace zookeeper-system -l "app.kubernetes.io/name=zookeeper,app.kubernetes.io/instance=my-zookeeper,app.kubernetes.io/component=zookeeper" -o jsonpath="{.items[0].metadata.name}")
kubectl exec -it $POD_NAME -- zkCli.sh
```

To connect to your ZooKeeper server from outside the cluster execute the following commands:

```
kubectl port-forward --namespace zookeeper-system svc/my-zookeeper 2181:2181 &
zkCli.sh 127.0.0.1:2181
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=local-hostpath, which is provided by base image openebs-localpv:2.11.0.
CMD helm install my-zookeeper --create-namespace --namespace zookeeper-system --set persistence.storageClass=local-hostpath bitnami/zookeeper --version 7.4.2
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/bitnami/zookeeper).