# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 3 StatefulSet replicas for kafka which requests "10Gi" storage.
* 1 StatefulSet replica for the zookeeper which requests "10Gi" storage.

## How to use it

Kafka can be accessed by consumers via port 9092 on the following DNS name from within your cluster:

```
my-kafka.kafka-system.svc.cluster.local
```

Each Kafka broker can be accessed by producers via port 9092 on the following DNS name(s) from within your cluster:

```
my-kafka-0.my-kafka-headless.kafka-system.svc.cluster.local:9092
```

To create a pod that you can use as a Kafka client run the following commands:

```
kubectl run my-kafka-client --restart='Never' --image docker.io/bitnami/kafka:2.8.0-debian-10-r61 --namespace kafka-system --command -- sleep infinity
kubectl exec --tty -i my-kafka-client --namespace kafka-system -- bash

PRODUCER:
    kafka-console-producer.sh \
        --broker-list my-kafka-0.my-kafka-headless.kafka-system.svc.cluster.local:9092 \
        --topic test

CONSUMER:
    kafka-console-consumer.sh \
        --bootstrap-server my-kafka.kafka-system.svc.cluster.local:9092 \
        --topic test \
        --from-beginning
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=local-hostpath, which is provided by base image openebs-localpv:2.11.0.
CMD helm install my-kafka --create-namespace --namespace kafka-system --set global.storageClass=local-hostpath bitnami/kafka --version 14.0.5
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/bitnami/kafka).