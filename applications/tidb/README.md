# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 1 TiDB operator deployment resource.
* 3 TiDB pd replicas which requests "20Gi" storage.
* 3 TiDB tikv replicas which requests "50Gi" storage.
* 2 tidb server using ClusterIP to expose service.
* deploy TiDB monitor using grafana and prometheus which requests "50Gi" storage.

## How to use it

### Access the database

By default, we use Cluster IP to expose the console service outside the k8s cluster.

Get the tidb Cluster IP:

```shell
kubectl get svc -n tidb-system | grep basic-tidb-cluster-tidb
basic-tidb-cluster-tidb        ClusterIP   10.96.3.124   <none>        4000/TCP,10080/TCP    15h
```

Access tidb database:

`mysql -h ${tidb_lb_ip} -p 4000 -u root`

`tidb_lb_ip` is the cluster IP of the TiDB service.

### Monitor

Access the Grafana monitoring dashboard,you can run the kubectl port-forward command to access the Grafana monitoring
dashboard,then open [http://localhost:3000](http://localhost:3000) in your browser and log on with the default username and password admin.

`kubectl port-forward -n tidb-system svc/basic-tidb-cluster-grafana 3000:3000 &>/tmp/portforward-grafana.log &`

Or access grafana dashboard with node port.

```shell
kubectl get svc -n tidb-system basic-tidb-cluster-grafana

NAME                         TYPE       CLUSTER-IP    EXTERNAL-IP   PORT(S)          AGE
basic-tidb-cluster-grafana   NodePort   10.96.0.225   <none>        3000:31180/TCP   18m
```

Access the prometheus monitoring data, run the kubectl port-forward command to access it. Open [http://localhost:9090](http://localhost:9000) in
your browser or access this address via a client tool.

`kubectl port-forward -n tidb-system svc/basic-tidb-cluster-prometheus 9090:9090 &>/tmp/portforward-prometheus.log &`

Or access prometheus server with node port.

```shell
kubectl get svc -n tidb-system basic-tidb-cluster-prometheus

NAME                            TYPE       CLUSTER-IP    EXTERNAL-IP   PORT(S)          AGE
basic-tidb-cluster-prometheus   NodePort   10.96.0.142   <none>        9090:31772/TCP   18m
```

## How to rebuild it

Modify manifest yaml file according to your needs, then run below command to rebuild it.

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://docs.pingcap.com/zh/tidb-in-kubernetes/stable).