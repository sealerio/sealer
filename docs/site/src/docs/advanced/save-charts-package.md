# Save helm chart package

Sealer support to save raw helm chart package to cloud image as oci format. with this feature, we can pull the helm
chart package in other offline production environment.

## Prerequisites

Prepare two nodes named the build node and the run node. At the same time need to install sealer and helm on it.

## Examples

### On the build node.

#### Start docker registry to save helm chart package.

start docker registry to transfer helm chart package to oci format.

```shell
docker run -p 5000:5000  --restart=always --name registry -v /registry/:/var/lib/registry -d registry
```

use helm push to save helm chart package to registry.

```shell
export HELM_EXPERIMENTAL_OCI=1
helm push mysql-8.8.25.tgz oci://localhost:5000/helm-charts
```

#### Use sealer build to save helm chart package from local registry to cloud image.

Prepare Kubefile:

```shell
[root@iZbp16ikro46xwgqzij67sZ build]# cat Kubefile 
FROM kubernetes:v1.19.8
COPY imageList manifests
```

Prepare imageList file:

```shell
[root@iZbp16ikro46xwgqzij67sZ build]# cat imageList 
localhost:5000/helm-charts/mysql:8.8.25
localhost:5000/helm-charts/nginx:9.8.0
```

Then run `sealer build -t my-kubernetes:v1.19.8 -f Kubefile .`and we can
use `sealer save my-kubernetes:v1.19.8 -o my-kubernetes.tar` to save the image to the local filesystem.

### On the run node.

load the image `my-kubernetes.tar` from the build node use `sealer load -i my-kubernetes.tar`.

#### Use sealer run to start the cluster

```shell
sealer run -d my-kubernetes:v1.19.8 -p password -m 172.16.0.230
```

#### Pull Helm chart on the run node.

When the cluster is up, we can pull the helm chart package use helm pull:

```shell
export HELM_EXPERIMENTAL_OCI=1
helm pull oci://sea.hub:5000/helm-charts/mysql --version 8.8.25
```
 