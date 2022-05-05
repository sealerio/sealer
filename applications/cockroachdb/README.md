# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 1 StatefulSet with 3 replicas for cassandra cluster which requests "80Gi" storage.

## How to use it

CockroachDB can be accessed via port 26257 at the following DNS name from within your cluster:

my-cockroachdb-public.cockroachdb-system.svc.cluster.local

Because CockroachDB supports the PostgreSQL wire protocol, you can connect to the cluster using any available PostgreSQL
client.

For example, you can open up an SQL shell to the cluster by running:

```
kubectl run -it --rm cockroach-client \
    --image=cockroachdb/cockroach \
    --restart=Never \
    --command -- \
    ./cockroach sql --insecure --host=my-cockroachdb-public.cockroachdb-system
```

From there, you can interact with the SQL shell as you would any other SQL shell, confident that any data you write will
be safe and available even if parts of your cluster fail.

Finally, to open up the CockroachDB admin UI, you can port-forward from your local machine into one of the instances in
the cluster:

```
kubectl port-forward my-cockroachdb-0 8080
```

Then you can access the admin UI at [http://localhost:8080/](http://localhost:8080/) in your web browser.

For more information on using CockroachDB, please see the project's docs at [CockroachDB official website](https://www.cockroachlabs.com/docs/).

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
# add helm repo and run helm install
RUN helm repo add cockroachdb https://charts.cockroachdb.com/
# set persistence.storageClass=local-hostpath, which is provided by base image openebs-localpv:2.11.0.
CMD helm install my-cockroachdb --create-namespace --namespace cockroachdb-system --set persistence.storageClass=local-hostpath cockroachdb/cockroachdb --version 6.0.9
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://artifacthub.io/packages/helm/cockroachdb/cockroachdb).