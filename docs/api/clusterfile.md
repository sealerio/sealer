# Clusterfile definition

```yaml
apiVersion: sealer.io/v1
kind: Cluster
metadata:
  name: my-cluster
  annotation:
    sealer.aliyun.com/etcd: "/data/etcd"
    sealer.aliyun.com/docker: "/var/lib/docker"
spec:
  image: my-kubernetes:v1.18.3 # name of ClusterImage
  env: # the cluster global ENV
  - DOMAIN="sealer.alibaba.com"
  provider: ALI_CLOUD # OR BAREMETAL , CONTAINER.
  ssh: # host ssh config
    passwd: xxx
    pk: xxx
    pkPasswd: xxx
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  containerRuntime:
    type: docker
  masters: # if provider is ALI_CLOUD or CONTAINER, you can specify the number of server, if BAREMETAL using ipList.
    cpu: 4
    memory: 8
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  # ipList:
  #  - 192.168.0.2
  #  - 192.168.0.3
  #  - 192.168.0.4
  nodes:
    cpu: 5
    memory: 8
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  # ipList:
  #  - 192.168.0.2
  #  - 192.168.0.3
  #  - 192.168.0.4

  status:
```

If you want change other configurations like using external etcd cluster, you can overwrite the default kubeadm config,

```yaml
FROM kubernetes:1.18.8
COPY my-kubeadm.yaml.tmp kubeadm.yaml.tmp
```

Clusterfile only care about the common configurations.

ENV and annotations can help you to extend your config values. sealer will render it in your yaml if you put then in manifest dir.
Also, sealer will generate a global.yaml witch contains all values, so if you're using helm, can use global config like this:

```shell script
# global will overwrite the default value
helm install chart name -f values.yaml -f global.yaml
```
