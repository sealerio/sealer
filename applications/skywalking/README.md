# Overview

SkyWalking is an Observability Analysis Platform and Application Performance Management system.
This image base on longhorn:v1.2.3  as its persistence storage engine.

Components included in this image:

* longhorn:v1.2.3
* elasticsearch:v6.8.6
* cluster require at least 30Gi for storage.

## how to use

* Make sure you have at least 3 node in your cluster for This image.
* ```kubectl port-forward svc/skywalking-ui 8080:80 --namespace skywalking ```

## How to rebuild it use helm
Kubefile:

```shell
FROM longhorn:v1.2.3

CMD helm repo add skywalking https://apache.jfrog.io/artifactory/skywalking-helm
CMD helm install skywalking skywalking/skywalking -n skywalking \
  --create-namespace \
  --set oap.image.tag=8.8.1 \
  --set oap.storageType=elasticsearch \
  --set ui.image.tag=8.8.1 \
  --set elasticsearch.imageTag=6.8.6 \
  --set elasticsearch.persistence.enabled=true \
  --set elasticsearch.priorityClassName=longhorn \
  --set elasticsearch.minimumMasterNodes=1
```

run below command to build it

```sealer build -t {Your Image Name} -f Kubefile -m cloud .```

More parameters see [official document here](https://github.com/apache/skywalking-kubernetes/tree/master/chart/skywalking)

