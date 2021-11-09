# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 1 DaemonSet for fluent-bit.
* 1 DaemonSet for node-exporter.
* 1 StatefulSet replica for loki server which requests "50Gi" storage.
* 1 Deployment replica for grafana.
* 1 Deployment replica for kube-state-metrics.
* 1 Deployment replica for alertmanager.
* 1 Deployment replica for pushgateway.
* 1 Deployment replica for prometheus-server.

## How to use it

Access the Grafana monitoring dashboard,you can run the kubectl port-forward command to access the Grafana monitoring
dashboard,then open `http://localhost:3000` in your browser and log on with the default username and password.

To get the admin user and password for the Grafana pod, run the following command:

```shell
kubectl get secret -n fluentbit-loki-stack-system  loki-grafana -o jsonpath="{.data.admin-user}" | base64 --decode ; echo
kubectl get secret -n fluentbit-loki-stack-system  loki-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
```

To access the Grafana UI, run the following command:

`kubectl port-forward -n fluentbit-loki-stack-system svc/loki-grafana 3000:80`

## How to rebuild it use helm

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
# add helm repo and run helm install
RUN helm repo add grafana https://grafana.github.io/helm-charts && helm repo update
# set persistence.storageClass=local-hostpath, which is provided by base image openebs-localpv:2.11.0.
CMD helm install --create-namespace --namespace fluentbit-loki-stack-system loki grafana/loki-stack --set fluent-bit.enabled=true,promtail.enabled=false,grafana.enabled=true,prometheus.enabled=true,prometheus.alertmanager.persistentVolume.enabled=false,prometheus.server.persistentVolume.enabled=false,loki.persistence.enabled=true,loki.persistence.storageClassName=local-hostpath,loki.persistence.size=50Gi
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://grafana.github.io/helm-charts)