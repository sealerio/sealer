# Build application image

## Motivations

Applications image contains applications with all dependencies except the base image, so applications image can install
to an already existing Kubernetes cluster. with applications image, cluster can be incremental updating, and we can
install applications to an already existing Kubernetes cluster.

## Use cases

### Build an application image

just add argument "--base=false", will build an application image. and the size of application image depends on the
docker image size in most cases. without rootfs,it will become slimmer.

```shell
sealer build -f Kubefile -t {your image name}:{tag} --base=false .
```

The image you built will not contain the embedded Kubernetes cluster. and it only contains all the docker image that
application needs, and contains helm chart or other operator manifests.

### Apply this application image

We can only support sealer cluster currently. Sealer hacked docker and registry to cache docker image in cluster, so if
cluster not installed by sealer, this app image will install failed.

```shell
sealer run {your image name}:{tag} -m x.x.x.x -p xxx.
```

Or using Clusterfile to overwrite the config files like helm values.