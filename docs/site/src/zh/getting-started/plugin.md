# 集群镜像插件使用

## 插件类型列表

### 主机名插件

主机名插件将帮助您更改所有主机名

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyHostname # 指定插件名称，将会转储到$rootfs/plugins目录下。
spec:
  type: HOSTNAME #插件类型
  action: PreInit # 指定运行阶段
  data: |
    192.168.0.2 master-0
    192.168.0.3 master-1
    192.168.0.4 master-2
    192.168.0.5 node-0
    192.168.0.6 node-1
    192.168.0.7 node-2
```

### 脚本插件

你可以在指定节点的任何阶段执行任何shell命令。

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyShell
spec:
  type: SHELL
  action: PostInstall # # 指定运行阶段【PreInit ｜ PostInstall ｜ PostClean】
  'on': node-role.kubernetes.io/master=
  data: |
    kubectl get nodes
```

```shell
action : [PreInit| PostInstall] # 指定执行shell的时机
  镜像挂载前阶段          |   action: Originally
  在初始化之前之前执行命令  |  action: PreInit
  在添加节点之前执行命令    |  action: PreJoin
  在添加节点之后执行命令    |  action: PostJoin
  在执行Kubefile CMD命令前 |   action: PreGuest
  在安装集群之后执行命令    |  action: PostInstall
  在清理集群前执行命令      |  action: PreClean
  在清理集群后执行命令      |  action: PostClean
  组合使用                | action: PreInit|PreJoin
on     : #指定执行命令的机器
  为空时默认在所有节点执行
  在所有master节点上执行  'on': master
  在所有node节点上执行    'on': node
  在指定IP上执行         'on': 192.168.56.113,192.168.56.114,192.168.56.115,192.168.56.116
  在有连续IP的机器上执行   'on': 192.168.56.113-192.168.56.116
  在指定label节点上执行(action需为PostInstall或PreClean)    'on': node-role.kubernetes.io/master=
data   : #指定执行的shell命令
  例:
    `data: echo "test shell plugin"`

  调用cluster env 执行脚本:
      执行shell 脚本命令: `data: echo $docker_dir $ips[@]`
      执行shell 脚本文件: `data: . scripts/install.sh` 或者 `data: source scripts/install.sh`
  ## 当sealer执行脚本时env的设置类似于：`docker_dir=/data/docker ips=(192.168.0.2;192.168.0.3) && source scripts/install.sh`
  ## data中使用bash scripts/install.sh的方式将开启新的进程执行脚本导致无法获取当前环境env
  ## 因此install.sh中如需使用env需要使用source scripts/install.sh
```

### 标签插件

帮助您在安装kubernetes集群后设置标签

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

### 集群检测插件

由于服务器以及环境因素(服务器磁盘性能差)可能会导致sealer安装完kubernetes集群后，立即部署应用服务，出现部署失败的情况。cluster check插件会等待kubernetes集群稳定后再部署应用服务。

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: checkCluster
spec:
  type: CLUSTERCHECK
  action: PreGuest
```

### 污点插件

如果你在Clusterfile后添加taint插件配置并应用它，sealer将帮助你添加污点和去污点：

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
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

> data写法为`ips taint_argument`
> ips           : 多个ip通过`,`连接，连续ip写法为 首ip-末尾ip
> taint_argument: 同kubernetes 添加或去污点写法(key=value:effect #effect必须为：NoSchedule, PreferNoSchedule 或 NoExecute)。

### Etcd 备份插件

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: MyBackup
spec:
  type: ETCD
  action: PostInstall
```

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

## 如何使用插件

### 通过Clusterfile使用插件

例如，在安装kubernetes集群后设置节点标签:

```yaml
apiVersion: sealer.cloud/v2
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

### 在Kubefile中使用默认插件

在很多情况下，可以不使用Clusterfile而使用插件，本质上是在使用插件之前存储了Clusterfile插件到$rootfs/plugins目录下 所以当我们构建镜像时可以添加自定义默认插件。

插件配置文件 shell.yaml:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
name: taint
spec:
type: SHELL
action: PostInstall
'on': node-role.kubernetes.io/master=
data: |
  kubectl get nodes
---
apiVersion: sealer.aliyun.com/v1alpha1
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

构建一个包含安装iscsi的插件(或更多插件)的集群镜像:

```shell script
sealer build -m lite -t kubernetes-iscsi:v1.19.8 .
```

通过镜像启动集群后插件也将被执行，而无需在Clusterfile中定义插件:
`sealer run kubernetes-iscsi:v1.19.8 -m x.x.x.x -p xxx`
