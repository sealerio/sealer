# Overview

This image collects Kubernetes manifests, [Grafana](http://grafana.com/) dashboards,
and [Prometheus rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) combined with
documentation and scripts to provide easy to operate end-to-end Kubernetes cluster monitoring
with [Prometheus](https://prometheus.io/) using the Prometheus Operator.

Components included in this image:

* The [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator)
* Highly available [Prometheus](https://prometheus.io/)
* Highly available [Alertmanager](https://github.com/prometheus/alertmanager)
* [Prometheus node-exporter](https://github.com/prometheus/node_exporter)
* [Prometheus Adapter for Kubernetes Metrics APIs](https://github.com/DirectXMan12/k8s-prometheus-adapter)
* [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics)
* [Grafana](https://grafana.com/)

This stack is meant for cluster monitoring, so it is pre-configured to collect metrics from all Kubernetes components.
In addition to that it delivers a default set of dashboards and alerting rules. Many of the useful dashboards and alerts
come from the [kubernetes-mixin project](https://github.com/kubernetes-monitoring/kubernetes-mixin), similar to this
project it provides composable jsonnet as a library for users to customize to their needs.

## About prometheus

We choose OpenEBS LocalPV as its default persistence storage and deploy it with two replicas resource. many prometheus rules in
this image will be loaded when prometheus start. you can check the`Prometheus` and `PrometheusRule` in the cluster for
more details.

```shell
kubectl get Prometheus -n kube-prometheus-stack-system
kubectl get PrometheusRule -n kube-prometheus-stack-system
```

## About grafana

We provide grafana as deployment resource in the cluster and which service type is `ClusterIP`. what's more, we hold
many useful grafana dashboard inside which will be applied with the grafana dashboard by default. you can check the
`ConfigMap`for more details in the `kube-prometheus-stack-system` namespace.

## Access the dashboard

Prometheus, Grafana, and Alertmanager dashboards can be accessed quickly using kubectl port-forward after running the
image via the commands below.

Examples:

```shell
kubectl --namespace kube-prometheus-stack-system port-forward service/kube-prometheus-stack-grafana 3000
```

Then access via [http://localhost:3000](http://localhost:3000) and use default dashboard access credential :

```shell
username : admin
password : sealer-admin
```
