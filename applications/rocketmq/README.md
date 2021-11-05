# Overview

This image chooses OpenEBS LocalPV as its persistence storage engine.

Components included in this image:

* 1 RocketMQ operator deployment resource.
* 1 RocketMQ console with port 8080 to provide web service.
* 2 RocketMQ broker replicas which requests "10Gi" storage.
* 1 RocketMQ nameserver deployment resource which requests "10Gi" storage.

## How to use it

By default, we use nodePort service to expose the console service outside the k8s cluster:

Then you can visit the RocketMQ Console (by default) by the URL any-k8s-node-IP:30000, or localhost:30000 if you are
currently on the k8s node.

## How to rebuild it

Modify manifest yaml file according to your needs, then run below command to rebuild it.

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More parameters see [official document here](https://github.com/apache/rocketmq-operator).