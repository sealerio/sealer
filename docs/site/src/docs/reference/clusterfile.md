# Clusterfile definition

Install to existing servers, the provider is `BAREMETAL`:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  provider: BAREMETAL
  ssh: # host ssh config
    # ssh login password. If you use the key, you don't need to set the password
    passwd:
    # The absolute path of the ssh private key file, for example, /root/.ssh/id_rsa
    pk: xxx
    # ssh private key file password
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  masters:
    ipList:
     - 172.20.125.1
     - 172.20.126.2
     - 172.20.126.3
  nodes:
    ipList:
     - 172.20.126.7
     - 172.20.126.8
     - 172.20.126.9
```

Automatically apply ali cloud server for installation, the provider is `ALI_CLOUD`. Or using container for installation，the provider is `CONTAINER`:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 # name of CloudImage
  provider: ALI_CLOUD # OR CONTAINER
  ssh: # custom host ssh config
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
  masters: # You can specify the number of servers, system disk, data disk, cpu and memory size
    cpu: 4
    memory: 8
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 5
    memory: 8
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  status: {}
```

The Clusterfile can apply with Plugins metadata and Configs metadata，when you need to [modify the configuration](../getting-started/using-plugin.md) or [use plugins](../getting-started/using-config.md).
