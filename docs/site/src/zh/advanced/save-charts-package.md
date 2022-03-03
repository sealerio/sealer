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

## Save ACR chart

Example to pull `chart-registry.cn-shanghai.cr.aliyuncs.com/aliyun-inc.com/elasticsearch:1.0.1-elasticsearch.elasticsearch` chart.

1. Login your ACR registry

```shell script
sealer login sealer login chart-registry.cn-shanghai.cr.aliyuncs.com \
   --username cnx-platform@prod.trusteeship.aliyunid.com --passwd xxx
```

2. Create Kubefile and imageList

```shell script
[root@iZ2zeasfsez3jrior15rpbZ chart]# cat imageList
chart-registry.cn-shanghai.cr.aliyuncs.com/aliyun-inc.com/elasticsearch:1.0.1-elasticsearch.elasticsearch
[root@iZ2zeasfsez3jrior15rpbZ chart]# cat Kubefile
FROM kubernetes:v1.19.8
COPY imageList manifests
```

3. Build CloudImage and save ACR remote chart to local registry

```shell script
sealer build -t chart:latest .
```

4. Run a cluster

```shell script
sealer run chart:latest -m x.x.x.x -p xxx
```

5. Try to pull chart using helm from local registry

```shell script
[root@iZ2zeasfsez3jrior15rpbZ certs]# helm pull oci://sea.hub:5000/aliyun-inc.com/elasticsearch --version 1.0.1-elasticsearch.elasticsearch
Warning: chart media type application/tar+gzip is deprecated
Pulled: sea.hub:5000/aliyun-inc.com/elasticsearch:1.0.1-elasticsearch.elasticsearch
Digest: sha256:c247fd56b985cfa4ad58c8697dc867a69ee1861a1a625b96a7b9d78ed5d9df95
[root@iZ2zeasfsez3jrior15rpbZ certs]# ls
elasticsearch-1.0.1-elasticsearch.elasticsearch.tgz
```

If you got `Error: failed to do request: Head "https://sea.hub:5000/v2/aliyun-inc.com/elasticsearch/manifests/1.0.1-elasticsearch.elasticsearch": x509: certificate signed by unknown authority
` error, trust registry cert on your host:

```shell script
cp /var/lib/sealer/data/my-cluster/certs/sea.hub.crt /etc/pki/ca-trust/source/anchors/ && update-ca-trust extract
```