# Build application image

## Motivations

Applications image contains applications with all dependencies except the base image, so applications image can install
to an already existing Kubernetes cluster. with applications image, cluster can be incremental updating, and we can
install applications to an already existing Kubernetes cluster.

## Use cases

### Build an application image

just add argument "--base=false", will build an application image. and the size of application image depends on the
docker image size in most cases. without rootfs,it will become slimmer.

For example to build a prometheus application image:
Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0
COPY prometheus manifests
CMD kubectl apply -f prometheus/crd.yaml
CMD kubectl apply -f prometheus/operator.yaml
```

build command:

```shell
sealer build -f Kubefile -t prometheus:2.30.0 --base=false .
```

The image prometheus:2.30.0 will not contain the embedded Kubernetes cluster. and it only contains all the docker image
that application needs, and contains helm chart or other operator manifests.

### Apply this application image

We can only support sealer cluster currently. Sealer hacked docker and registry to cache docker image in cluster, so if
cluster not installed by sealer, this app image will install failed.

```shell
sealer run prometheus:2.30.0
```

Or using Clusterfile to overwrite the config files like helm values.

### Merge two application images

Using sealer merge ,we can combine the app image as one,in the meantime we can merge application images with cloud
images.

```shell
sealer merge mysql:8.0.26 redis:6.2.5 -t mysql-redis:latest
```
