# Overview

This image chooses OpenEBS Jiva as its persistence storage engine.

Components included in this image:

* 1 StatefulSet replica for mysql primary which requests "80Gi" storage.
* 1 StatefulSet replica for mysql secondary which requests "80Gi" storage.

## How to use it

Tip:

Watch the deployment status using the command: kubectl get pods -w --namespace mysql-system

Services:

echo Primary: my-mysql-primary.mysql-system.svc.cluster.local:3306 echo Secondary:
my-mysql-secondary.mysql-system.svc.cluster.local:3306

Administrator credentials:

echo Username: root echo Password : $(kubectl get secret --namespace mysql-system my-mysql -o jsonpath="
{.data.mysql-root-password}" | base64 --decode)

To connect to your database:

1. Run a pod that you can use as a client:

   kubectl run my-mysql-client --rm --tty -i --restart='Never' --image docker.io/bitnami/mysql:8.0.26-debian-10-r10
   --namespace mysql-system --command -- bash

2. To connect to primary service (read/write):

   mysql -h my-mysql-primary.mysql-system.svc.cluster.local -uroot -p my_database

3. To connect to secondary service (read-only):

   mysql -h my-mysql-secondary.mysql-system.svc.cluster.local -uroot -p my_database

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=openebs-jiva-csi-sc, which is provided by base image openebs-jiva:2.11.0.
CMD helm install my-mysql --set architecture=replication --create-namespace --namespace mysql-system --set secondary.persistence.storageClass=openebs-jiva-csi-sc --set primary.persistence.storageClass=openebs-jiva-csi-sc --set secondary.replicaCount=1 bitnami/mysql --version 8.8.6
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [officail document here](https://artifacthub.io/packages/helm/bitnami/mysql).