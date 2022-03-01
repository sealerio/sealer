# sealer registry configuration

The dependent images will be cached in the cluster images during the creation of the cluster images,
and Registry containing the cached data will be started when the cluster is installed through the cluster images

## Customize the config file to configure the cluster Registry:

Clusterfile:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  provider: BAREMETAL
...
...
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: registry_config
spec:
  path: etc/registry_config.yml
  data: |
    version: 0.1
    log:
      fields:
        service: registry
    storage:
      cache:
        blobdescriptor: inmemory
      filesystem:
        rootdirectory: /var/lib/registry
    http:
      addr: :5000
      headers:
        X-Content-Type-Options: [nosniff]
    proxy:
      on: true
    health:
      storagedriver:
        enabled: true
        interval: 10s
        threshold: 3
```

```shell
#sealer will write data from the data to '$rootfs/etc/registry_config.yml' file before registry starts. When to start the registry will mount the file to the registry ` config file/etc/docker/registry/config. Yml `.
#example: docker run ... -v $rootfs/etc/registry_config.yml:/etc/docker/registry/config.yml registry:2.7.1
sealer apply -f Clusterfile
```

## registry custom domain, port, username and password:

Clusterfile:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  provider: BAREMETAL
...
...
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: registry_passwd
spec:
  path: etc/registry.yml
  data: |
    domain: sea.hub
    port: "5000"
    username: sealerUser
    password: sealerPWD
```

```shell
# Sealer will generate the encrypted password for this authentication and write to the '$rootfs/etc/registry_htpasswd' file, which will be mounted and set to htpasswd authentication when Registry starts.
#docker run ... \
#        -v $rootfs/etc/registry_htpasswd:/htpasswd \
#        -e REGISTRY_AUTH=htpasswd \
#        -e REGISTRY_AUTH_HTPASSWD_PATH=/htpasswd \
#        -e REGISTRY_AUTH_HTPASSWD_REALM="Registry Realm" registry:2.7.1
sealer apply -f Clusterfile
```