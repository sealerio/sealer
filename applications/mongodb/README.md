# Overview

This image chooses OpenEBS Jiva as its persistence storage engine.

Components included in this image:

* 1 StatefulSet with 2 replicas for mongodb which requests "50Gi" storage.
* 1 StatefulSet with 1 replica for mongodb arbiter.

## How to use it

MongoDB&reg; can be accessed on the following DNS name(s) and ports from within your cluster:

```
my-mongodb-0.my-mongodb-headless.mongodb-system.svc.cluster.local:27017
my-mongodb-1.my-mongodb-headless.mongodb-system.svc.cluster.local:27017
```

To get the root password run:

```
export MONGODB_ROOT_PASSWORD=$(kubectl get secret --namespace mongodb-system my-mongodb -o jsonpath="{.data.mongodb-root-password}" | base64 --decode)
```

To connect to your database, create a MongoDB&reg; client container:

```
kubectl run --namespace mongodb-system my-mongodb-client --rm --tty -i --restart='Never' --env="MONGODB_ROOT_PASSWORD=$MONGODB_ROOT_PASSWORD" --image docker.io/bitnami/mongodb:4.4.8-debian-10-r9 --command -- bash
```

Then, run the following command:

```
mongo admin --host "my-mongodb-0.my-mongodb-headless.mongodb-system.svc.cluster.local:
27017,my-mongodb-1.my-mongodb-headless.mongodb-system.svc.cluster.local:27017" --authenticationDatabase admin -u root -p
$MONGODB_ROOT_PASSWORD
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=openebs-jiva-csi-sc, which is provided by base image openebs-jiva:2.11.0.
CMD helm install my-mongodb --set architecture=replication --create-namespace --namespace mongodb-system --set global.storageClass=openebs-jiva-csi-sc bitnami/mongodb --version 10.25.1
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/bitnami/mongodb).