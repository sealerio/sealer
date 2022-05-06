# Overview

This image is based on kubernetes:v1.19.9,  add longhorn  to provide persistent-volume.

Components included in this image:

* Enterprise-grade distributed storage with no single point of failure
* Incremental snapshot of block storage
* Backup to secondary storage (NFSv4 or S3-compatible object storage) built on efficient change block detection
* Recurring snapshot and backup
* Automated non-disruptive upgrade. You can upgrade the entire Longhorn software stack without disrupting running volumes!
* Intuitive GUI dashboard

## How to use it

At the first maker sure all the pods' status up in longhorn-system namespace.

### Storage class

These images provide  ```longhorn``` as default storage class, you can use it easily in PVC.

### UI dashboard

Once Longhorn has been installed in your Kubernetes cluster, you can access the UI dashboard.

1. Get the Longhorn’s external service IP:

​        ``` kubectl -n longhorn-system get svc```

​        For Longhorn v0.8.0, the output should look like this, and the `CLUSTER-IP` of the `longhorn-frontend` is used to access the Longhorn UI:

2. Navigate to the IP of `longhorn-frontend` in your browser.

## How to rebuild it use helm

Kubefile:

```yaml
helm repo add longhorn https://charts.longhorn.io
helm repo update
helm install longhorn longhorn/longhorn --namespace longhorn-system --create-namespace
```

run below command to build it

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://longhorn.io/docs/1.2.3/).
