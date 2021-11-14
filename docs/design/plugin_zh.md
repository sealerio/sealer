# 集群镜像 plugin

插件可以帮助用户做一些之外的事情，比如更改主机名，升级内核，或者添加节点标签等……

## 主机名 plugin

如果你在Clusterfile后添加插件配置并应用它，sealer将帮助你更改所有的主机名：

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: hostname
spec:
  type: HOSTNAME
  data: |
    192.168.0.2 master-0
    192.168.0.3 master-1
    192.168.0.4 master-2
    192.168.0.5 node-0
    192.168.0.6 node-1
    192.168.0.7 node-2
```

> Hostname Plugin 将各个节点在安装集群前修改为对应的主机名。

## shell plugin
如果你在Clusterfile后添加Shell插件配置并应用它，sealer将帮助你执行shell命令(执行路径为镜像Rootfs目录)：

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: shell
spec:
  type: SHELL
  action: PostInstall
  on: role=master
  data: |
    kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule
```

```shell
action : [PreInit| PreInstall| PostInstall] # 指定执行shell的时机
  在初始化之前之前执行命令  action: PreInit
  在安装集群之前执行命令    action: PreInstall
  在安装集群之后执行命令    action: PostInstall
on     : #指定执行命令的机器
  在所有master上执行    on: role=master
  在所有node上执行      on: role=node
  在指定IP上执行        on: 192.168.56.113,192.168.56.114,192.168.56.115,192.168.56.116
  在有连续IP的机器上执行  on: 192.168.56.113-192.168.56.116
data   : #指定执行的shell命令
```

## label plugin


如果你在Clusterfile后添加label插件配置并应用它，sealer将帮助你添加label：

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: label
spec:
  type: LABEL
  data: |
    192.168.0.2 ssd=true
    192.168.0.3 ssd=true
    192.168.0.4 ssd=true
    192.168.0.5 ssd=false,hdd=true
    192.168.0.6 ssd=false,hdd=true
    192.168.0.7 ssd=false,hdd=true
```

> 节点ip与标签之前使用空格隔开，多个标签之间使用逗号隔开。

## clusterCheck plugin

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


## plugin使用步骤:

Clusterfile内容：

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  provider: BAREMETAL
  ssh:
    # ssh的私钥文件绝对路径，例如/root/.ssh/id_rsa
    pk: xxx
    # ssh的私钥文件密码，如果没有的话就设置为""
    pkPasswd: xxx
    # ssh登录用户
    user: root
    # ssh的登录密码，如果使用的密钥登录则无需设置
    passwd: xxx
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  masters:
    ipList:
     - 192.168.0.2
     - 192.168.0.3
     - 192.168.0.4
  nodes:
    ipList:
     - 192.168.0.5
     - 192.168.0.6
     - 192.168.0.7
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: hostname
spec:
  type: HOSTNAME
  data: |
     192.168.0.2 master-0
     192.168.0.3 master-1
     192.168.0.4 master-2
     192.168.0.5 node-0
     192.168.0.6 node-1
     192.168.0.7 node-2
---
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
```

```
sealer apply -f Clusterfile #plugin仅在安装时执行，后续apply不生效。
```

> 执行上述命令后hostname，shell plugin将修改主机名并在成功安装集群后执行shell命令。

## 在Kubefile中定义默认插件

很多情况下在不使用Clusterfile的情况下也能使用插件，本质上sealer会先把Clusterfile中的插件配置先存储到 rootfs/plugin目录，再去使用，所以我们可以在制作镜像时就定义好默认插件。

插件配置 shell.yaml:

```
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
```

Kubefile:

```shell script
FROM kubernetes:v1.19.8
COPY shell.yaml plugin
```

build一个包含去污点插件的集群镜像：

```shell script
sealer build -b lite -t kubernetes-taint:v1.19.8 .
```

后续直接run这个镜像，插件也会被执行，而不再需要在Clusterfile中定义插件：`sealer run kubernetes-taint:v1.19.8 -m x.x.x.x -p xxx`
