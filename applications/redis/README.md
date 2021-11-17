# Overview

This image chooses OpenEBS Jiva as its persistence storage engine.

Components included in this image:

* 1 StatefulSet with 3 replicas for redis cluster which requests "50Gi" storage.

## How to use it

Redis&trade; can be accessed via port 6379 on the following DNS name from within your cluster:

```
my-myredis.redis-system.svc.cluster.local for read only operations
```

For read/write operations, first access the Redis&trade; Sentinel cluster, which is available in port 26379 using the
same domain name above.

To get your password run:

```
export REDIS_PASSWORD=$(kubectl get secret --namespace redis-system my-myredis -o jsonpath="{.data.redis-password}" | base64 --decode)
```

To connect to your Redis&trade; server:

1. Run a Redis&trade; pod that you can use as a client:

```
kubectl run --namespace redis-system redis-client --restart='Never'  --env REDIS_PASSWORD=$REDIS_PASSWORD --image docker.io/bitnami/redis:6.2.5-debian-10-r11 --command -- sleep infinity
```

Use the following command to attach to the pod:

```
kubectl exec --tty -i redis-client --namespace redis-system -- bash
```

2. Connect using the Redis&trade; CLI:

```
redis-cli -h my-myredis -p 6379 -a $REDIS_PASSWORD # Read only operations redis-cli -h my-myredis -p 26379 -a $REDIS_PASSWORD # Sentinel access
```

To connect to your database from outside the cluster execute the following commands:

```
kubectl port-forward --namespace redis-system svc/my-myredis 6379:6379 &
redis-cli -h 127.0.0.1 -p 6379 -a $REDIS_PASSWORD
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=openebs-jiva-csi-sc, which is provided by base image openebs-jiva:2.11.0.
CMD helm install my-redis --create-namespace --namespace redis-system --set master.persistence.storageClass=openebs-jiva-csi-sc --set replica.persistence.storageClass={Your Stroage Class} --set sentinel.enabled=true bitnami/redis --version 15.3.0
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/bitnami/redis).