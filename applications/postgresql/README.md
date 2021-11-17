# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 1 StatefulSet replica for postgresql primary which requests "50Gi" storage.
* 1 StatefulSet replica for postgresql read which requests "50Gi" storage.

## How to use it

PostgreSQL can be accessed via port 5432 on the following DNS names from within your cluster:

```
my-postgresql.postgresql-system.svc.cluster.local - Read/Write connection
my-postgresql-read.postgresql-system.svc.cluster.local - Read only connection
```

To get the password for "postgres" run:

```
export POSTGRES_PASSWORD=$(kubectl get secret --namespace postgresql-system my-postgresql -o jsonpath="{.data.postgresql-password}" | base64 --decode)
```

To connect to your database run the following command:

```
kubectl run my-postgresql-client --rm --tty -i --restart='Never' --namespace postgresql-system --image docker.io/bitnami/postgresql:11.12.0-debian-10-r70 --env="PGPASSWORD=$POSTGRES_PASSWORD" --command -- psql --host my-postgresql -U postgres -d postgres -p 5432
```

To connect to your database from outside the cluster execute the following commands:

```
kubectl port-forward --namespace postgresql-system svc/my-postgresql 5432:5432 & PGPASSWORD="$POSTGRES_PASSWORD" psql --host 127.0.0.1 -U postgres -d postgres -p 5432
```

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
# set persistence.storageClass=local-hostpath, which is provided by base image openebs-localpv:2.11.0.
CMD helm install my-postgresql --create-namespace --namespace postgresql-system --set replication.enabled=true --set global.storageClass=local-hostpath bitnami/postgresql --version 10.9.4
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/bitnami/postgresql).