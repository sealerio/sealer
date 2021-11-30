# Plugin Usage

## Plugin type list

### hostname plugin

HOSTNAME plugin will help you to change all the hostnames

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyHostname # Specify this plugin name,will dump in $rootfs/plugin dir.
spec:
  type: HOSTNAME # fixed string,should not change this name.
  action: PreInit # Specify which phase to run.
  data: |
    192.168.0.2 master-0
    192.168.0.3 master-1
    192.168.0.4 master-2
    192.168.0.5 node-0
    192.168.0.6 node-1
    192.168.0.7 node-2
```

### shell plugin

You can exec any shell command on specify node in any phase.

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyShell # Specify this plugin name,will dump in $rootfs/plugin dir.
spec:
  type: SHELL
  action: PostInstall # PreInit PreInstall PostInstall
  data: |
    kubectl get nodes
```

action: the phase of command.

* PreInit: before init master0.
* PreInstall: before join master and nodes.
* PostInstall: after join all nodes.

on: exec on which node.

### label plugin

Help you set label after install kubernetes cluster

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyLabel
spec:
  type: LABEL
  action: PostInstall
  data: |
    192.168.0.2 ssd=true
    192.168.0.3 ssd=true
    192.168.0.4 ssd=true
    192.168.0.5 ssd=false,hdd=true
    192.168.0.6 ssd=false,hdd=true
    192.168.0.7 ssd=false,hdd=true
```

### Etcd backup plugin

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyBackup
spec:
  type: ETCD
  action: PostInstall
```

Etcd backup plugin is triggered manually: `sealer plugin -f etcd_backup.yaml`

### Out of tree plugin

at present, we only support the golang so file as out of tree plugin. More description about golang plugin
see [golang plugin website](https://pkg.go.dev/plugin).

copy the so file and the plugin config to your cloud image at build stage use `Kubefile`,sealer will parse and execute
it. develop your own out of tree plugin see [sealer plugin](../advanced/develop-plugin.md).

plugin config:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: label_nodes.so # out of tree plugin name
spec:
  type: LABEL_TEST_SO # define your own plugin type.
  action: PostInstall # which stage will this plugin be applied.
  data: |
    192.168.0.2 ssd=true
```

Kubefile:

```shell script
FROM kubernetes:v1.19.8
COPY label_nodes.so plugin
COPY label_nodes.yaml plugin
```

Build a cluster image that contains the golang plugin (or more plugins):

```shell script
sealer build -m lite -t kubernetes-post-install:v1.19.8 .
```

## How to use plugin

### use it via Clusterfile

For example, set node label after install kubernetes cluster:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  provider: BAREMETAL
  ssh:
    passwd:
    pk: xxx
    pkPasswd: xxx
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2

  masters:
    ipList:
      - 172.20.126.4
      - 172.20.126.5
      - 172.20.126.6
  nodes:
    ipList:
      - 172.20.126.8
      - 172.20.126.9
      - 172.20.126.10
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: LABEL
spec:
  type: LABEL
  action: PostInstall
  data: |
    172.20.126.8 ssd=false,hdd=true
```

```shell script
sealer apply -f Clusterfile
```

### use it via Kubefile

Define the default plugin in Kubefile to build the image and run it.

In many cases it is possible to use plugins without using Clusterfile, essentially sealer stores the Clusterfile plugin
configuration in the Rootfs/Plugin directory before using it, so we can define the default plugin when we build the
image.

Plugin configuration shell.yaml:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
name: taint
spec:
type: SHELL
action: PostInstall
on: role=master
data: |
  kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=node
  data: |
    if type yum >/dev/null 2>&1;then
    yum -y install iscsi-initiator-utils
    systemctl enable iscsid
    systemctl start iscsid
    elif type apt-get >/dev/null 2>&1;then
    apt-get update
    apt-get -y install open-iscsi
    systemctl enable iscsid
    systemctl start iscsid
    fi
```

Kubefile:

```shell script
FROM kubernetes:v1.19.8
COPY shell.yaml plugin
```

Build a cluster image that contains a taint plugin (or more plugins):

```shell script
sealer build -m lite -t kubernetes-taint:v1.19.8 .
```

Run the image and the plugin will also be executed without having to define the plug-in in the Clusterfile:
`sealer run kubernetes-taint:v1.19.8 -m x.x.x.x -p xxx`
