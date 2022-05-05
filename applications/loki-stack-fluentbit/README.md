# Overview

## Introduction of components

Loki is a horizontally-scalable, highly-available, multi-tenant log aggregation system inspired by [Prometheus](https://prometheus.io/). It is designed to be very cost-effective and easy to operate. It does not index the contents of the logs, but rather a set of labels for each log stream.

Fluent Bit is a fast Log Processor and Forwarder for Linux, Windows, Embedded Linux, macOS and BSD family operating systems. It's part of the Graduated Fluentd Ecosystem and a CNCF sub-project.

This image base on  longhorn:latest, flunent-bit can collect logs and longhorn provide storage to save logs in persistent volume.

The fluentbit-loki-stack component include:

* 1 DaemonSet for fluent-bit.
* 1 DaemonSet for node-exporter.
* 1 StatefulSet replica for loki server which requests "50Gi" storage.
* 1 Deployment replica for grafana.
* 1 Deployment replica for kube-state-metrics.
* 1 Deployment replica for alertmanager.
* 1 Deployment replica for pushgateway.
* 1 Deployment replica for prometheus-server.

## How to use it

### Browser the dashboard

* You can browser it in the cluster.

  1. Run ``kubectl get svc -n fluentbit-loki-stack-system | grep grafana`` To get pod IP address.

  ```
     # kubectl get svc -n fluentbit-loki-stack-system | grep grafana
     fluentbit-loki-stack-grafana                    ClusterIP   10.96.1.50    <none>        80/TCP     37s
  ```

  2. you can easily browser the dashboard in you cluster network.

* You can forward the container port to NodePort.

  1. Run this command ``kubectl port-forward -n fluentbit-loki-stack-system svc/loki-grafana 3000:80``
  2. Then you can browser the dashboard from outside of cluster network.

### Get the user-name and password

* you should be required to input username and password when you access to the dashboard.

* you can get the  username with these command.

  ```
  kubectl get secret -n fluentbit-loki-stack-system fluentbit-loki-stack-grafana -o jsonpath="{.data.admin-user}" | base64 --decode ; echo
  kubectl get secret -n fluentbit-loki-stack-system fluentbit-loki-stack-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
  ```

## How to rebuild it use helm

Kubefile:

```
FROM longhorn:latest

CMD helm repo add grafana https://grafana.github.io/helm-charts
CMD helm repo update

CMD helm install fluentbit-loki-stack grafana/loki-stack \
 	--create-namespace --namespace fluentbit-loki-stack-system \
	--set fluent-bit.enabled=true \
	--set promtail.enabled=false \
	--set grafana.enabled=true \
	--set prometheus.enabled=true \
	--set prometheus.alertmanager.persistentVolume.enabled=false \
	--set prometheus.server.persistentVolume.enabled=false \
	--set loki.persistence.enabled=true \
	--set loki.persistence.storageClassName=longhorn \
	--set loki.persistence.size=50Gi
```

run below command to build it

```
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://longhorn.io/docs/1.2.3/).
