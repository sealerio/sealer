# Overview

This image base on longhorn:v1.2.3 as its persistence storage engine.

Components included in this image:

- 1 DaemonSet for promtail.
- 1 DaemonSet for node-exporter.
- 1 StatefulSet replica for loki server which requests "50Gi" storage.
- 1 Deployment replica for grafana.
- 1 Deployment replica for kube-state-metrics.
- 1 Deployment replica for alertmanager.
- 1 Deployment replica for pushgateway.
- 1 Deployment replica for prometheus-server.

## How to use it

### Browser the dashboard

- 1 Prometheus, Grafana, and Alertmanager dashboards can be accessed quickly using kubectl port-forward after running the quickstart via the commands below.

- 1 Prometheus

  ```
  kubectl --namespace kube-prometheus-stack port-forward svc/kube-prometheus-stack-prometheus 9090
  ```

  Then access via [http://localhost:9090](http://localhost:9090/)

  Grafana

  ```
  kubectl --namespace kube-prometheus-stack port-forward svc/kube-prometheus-stack-grafana 3000
  ```

  Then access via [http://localhost:3000](http://localhost:3000/) and use the default grafana user:password of `admin:admin`.

  Alert Manager

  ```
  kubectl --namespace kube-prometheus-stack port-forward svc/kube-prometheus-stack-alertmanager 9093
  ```

  Then access via [http://localhost:9093](http://localhost:9093/)

  ```
  kubectl --namespace kube-prometheus-stack port-forward service/kube-prometheus-stack-grafana 80
  ```

#### Get the user-name and password

- 1 you should be required to input username and password when you access to the dashboard.

- 1 you can get the  username with these command.

  ```shell
  kubectl get secret -n fluentbit-loki-stack-system fluentbit-loki-stack-grafana -o jsonpath="{.data.admin-user}" | base64 --decode ; echo
  kubectl get secret -n fluentbit-loki-stack-system fluentbit-loki-stack-grafana -o jsonpath="{.data.admin-password}" | base64 --decode ; echo
  ```

## How to rebuild it use helm

```shell
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install kube-prometheus-stack prometheus-community/kube-prometheus-stack \
    --namespace kube-prometheus-stack --create-namespace \
    --set prometheus.alertmanager.persistentVolume.enabled=false \
    --set prometheus.server.persistentVolume.enabled=false
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack).
