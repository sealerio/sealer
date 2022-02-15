# 集群镜像 registry 配置

集群镜像在制作时将依赖的镜像缓存在集群镜像之中，通过集群镜像安装集群时将启动包含镜像缓存数据的registry

## 自定义config文件配置集群registry:

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
#sealer将会在registry启动前将data中的数据写入到`$rootfs/etc/registry_config.yml`文件，在启动registry时将该文件挂载到registry的config文件`/etc/docker/registry/config.yml`。
sealer apply -f Clusterfile
```

## 自定义registry域名，端口，用户名及密码：

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
#sealer将生成该认证的加密密码并写入`$rootfs/etc/registry_htpasswd`文件，在registry启动时将会挂载该文件并设置认证为htpasswd。
sealer apply -f Clusterfile
```