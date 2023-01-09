# Plugin Usage

## Plugin type list

### hostname plugin

HOSTNAME plugin will help you to change all the hostnames

```yaml
---
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyHostname # Specify this plugin name,will dump in $rootfs/plugins dir.
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
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyShell # Specify this plugin name,will dump in $rootfs/plugins dir.
spec:
  type: SHELL
  action: PostInstall # PreInit PostInstall
  data: |
    kubectl get nodes
```

```shell
action : [PreInit| PostInstall] # Specify phases to execute the shell
  Pre-initialization phase            |   action: PreInit
  Pre-join phase                      |   action: PreJoin
  Post-join phase                     |   action: PostJoin
  before exec Kubefile CMD phase      |   action: PreGuest
  after  installing the cluster phase |   action: PostInstall
  before clean cluster phase          |   action: PreClean
  after clean cluster phase           |   action: PostClean
  combined use phase                  |   action: PreInit|PreJoin
on     : #Specifies the machine to execute the command
  If null, it is executed on all nodes by default
  on all master nodes                 |  'on': master
  on all work nodes                   |  'on': node
  on the specified IP address         |  'on': 192.168.56.113,192.168.56.114,192.168.56.115,192.168.56.116
  on a machine with continuous IP     |  'on': 192.168.56.113-192.168.56.116
  on the specified label node (action must be PostInstall or PreClean)  |  'on': node-role.kubernetes.io/master=
data   : #Specifies the shell command to execute
```

### label plugin

Help you set label after install kubernetes cluster

```yaml
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyLabel
spec:
  type: LABEL
  action: PreGuest
  data: |
    192.168.0.2 ssd=true
    192.168.0.3 ssd=true
    192.168.0.4 ssd=true
    192.168.0.5 ssd=false,hdd=true
    192.168.0.6 ssd=false,hdd=true
    192.168.0.7 ssd=false,hdd=true
```

## clusterCheck plugin

Server and environmental factors (poor server disk performance) may cause Sealer to deploy the application services immediately after installing the Kubernetes cluster, causing deployment failures.
The Cluster Check plugin waits for the Kubernetes cluster to stabilize before deploying the application service.

```yaml
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: checkCluster
spec:
  type: CLUSTERCHECK
  action: PreGuest
```

### Etcd backup plugin

```yaml
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: MyBackup
spec:
  type: ETCD
  action: PostInstall
```

Etcd backup plugin is triggered manually: `sealer plugin -f etcd_backup.yaml`

### taint plugin

Add or remove taint by adding the taint plugin for the PreGuest phase:

```yaml
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: taint
spec:
  type: TAINT
  action: PreGuest
  data: |
    192.168.56.3 key1=value1:NoSchedule
    192.168.56.4 key2=value2:NoSchedule-
    192.168.56.3-192.168.56.7 key3:NoSchedule
    192.168.56.3,192.168.56.4,192.168.56.5,192.168.56.6,192.168.56.7 key4:NoSchedule
    192.168.56.3 key5=:NoSchedule
    192.168.56.3 key6:NoSchedule-
    192.168.56.4 key7:NoSchedule-
```

> The value of data is `ips taint_argument`,
> ips: Multiple IP addresses are connected through ',', and consecutive IP addresses are written as the first IP address and the last IP address;
> taint_argument: Same as kubernetes add or remove taints writing (key=value:effect #The effect must be NoSchedule, PreferNoSchedule or NoExecute)。

### Out of tree plugin

at present, we only support the golang so file as out of tree plugin. More description about golang plugin
see [golang plugin website](https://pkg.go.dev/plugin).

copy the so file and the plugin config to your ClusterImage at build stage use `Kubefile`,sealer will parse and execute
it.

plugin config:

```yaml
apiVersion: sealer.io/v1
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

Build a ClusterImage that contains the golang plugin (or more plugins):

```shell script
sealer build -m lite -t kubernetes-post-install:v1.19.8 .
```

## How to use plugin

### use it via Clusterfile

For example, set node label after install kubernetes cluster:

```yaml
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
  ssh:
    passwd: xxx
  hosts:
    - ips: [ 192.168.0.2,192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
---
apiVersion: sealer.io/v1
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
configuration in the Rootfs/Plugins directory before using it, so we can define the default plugin when we build the
image.

Plugin configuration shell.yaml:

```yaml
apiVersion: sealer.io/v1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
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

Build a ClusterImage that contains an installation iscsi plugin (or more plugins):

```shell script
sealer build -m lite -t kubernetes-iscsi:v1.19.8 .
```

Run the image and the plugin will also be executed without having to define the plugin in the Clusterfile:
`sealer run kubernetes-iscsi:v1.19.8 -m x.x.x.x -p xxx`
