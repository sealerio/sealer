# Sealer

## 什么是sealer

sealer[ˈsiːlər]是一款分布式应用打包交付运行的解决方案，通过把分布式应用及其数据库中间件等依赖一起打包以解决复杂应用的交付问题。
sealer构建出来的产物我们称之为"集群镜像", 集群镜像里内嵌了一个kubernetes, 解决了分布式应用的交付一致性问题。
集群镜像可以push到registry中共享给其他用户使用，也可以在官方仓库中找到非常通用的分布式软件直接使用。

Docker可以把一个操作系统的rootfs+应用 build成一个容器镜像，sealer把kubernetes看成操作系统，在这个更高的抽象纬度上做出来的镜像就是集群镜像。
实现整个集群的Build Share Run !!!

有了集群镜像用户实践云原生生态技术将变得极其简单，如：

> 安装一个kubernetes集群

```shell script
#安装sealer
wget https://github.com/alibaba/sealer/releases/download/v0.5.0/sealer-v0.5.0-linux-amd64.tar.gz && \
tar zxvf sealer-v0.5.0-linux-amd64.tar.gz && mv sealer /usr/bin
#运行集群（安装完成后生成`/root/.sealer/[cluster-name]/Clusterfile`文件用于存放集群相关信息）
sealer run kubernetes:v1.19.8 # 在公有云上运行一个kubernetes集群
sealer run kubernetes:v1.19.8 --masters 3 --nodes 3 # 在公有云上运行指定数量节点的kuberentes集群
# 安装到已经存在的机器上
sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 --nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
```

> 安装prometheus集群

```shell script
sealer run prometheus:2.26.0
```

上面命令就可以帮助你安装一个包含promeheus的kubernetes集群, 同理其它软件如istio ingress grafana等都可以通过这种方式运行。

还没完，Sealer最出色的地方是可以非常方便的让用户自定义一个集群的镜像，通过像Dockerfile一样的文件来描述和build：

Kubefile:

```shell script
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest .
```

然后一个包含dashboard的集群镜像就被制作出来了，可以运行或者分享给别人。

Sealer提供提供三种build方式，默认为cloud模式 : [Build使用文档](build/build_zh.md)

把制作好的集群镜像推送到镜像仓库，集群镜像仓库兼容docker镜像仓库标准，可以把集群镜像推送到docker hub、阿里ACR、或者Harbor中

```shell script
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

## 使用场景&特性

- [x] 极其简单的方式在生产环境中或者离线环境中安装kubernetes、以及kubernetes生态中其它软件
- [x] 通过Kubefile可以非常简单的自定义kubernetes集群镜像对集群和应用进行打包，并可以提交到仓库中进行分享
- [x] 强大的生命周期管理能力，以难以想象的简单的方式去做如集群升级，集群备份恢复，节点扩缩等操作
- [x] 速度极快3min以内完成集群安装
- [x] 支持ARM x86, v1.20以上版本支持containerd，几乎兼容所有支持systemd的linux操作系统
- [x] 不依赖ansible haproxy keepalived, 高可用通过ipvs实现，占用资源少，稳定可靠
- [x] 官方仓库中有非常多的生态软件镜像可以直接使用，包含所有依赖，一键安装

## 快速开始

## 安装一个kubernetes集群

```shell script
sealer run kubernetes:v1.19.8 --masters 192.168.0.2 --passwd xxx #sealer使用内置docker实现镜像缓存功能，安装节点不能使用自带docker
```

如果是在云上安装（需设置阿里云[AK，SK](https://ram.console.aliyun.com/manage/ak) ）:

```shell script
export ACCESSKEYID=xxx
export ACCESSKEYSECRET=xxx
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
# 或者指定节点数量运行集群
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest \
  --masters 3 --nodes 3
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                      STATUS   ROLES    AGE   VERSION
izm5e42unzb79kod55hehvz   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r7z   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r8z   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r9z   Ready    <none>   18h   v1.16.9
izm5ehdjw3kru84f0kq7raz   Ready    <none>   18h   v1.16.9
izm5ehdjw3kru84f0kq7rbz   Ready    <none>   18h   v1.16.9
```

run命令使用镜像默认配置Clusterfile安装集群，可使用`sealer inspect [镜像名称] -c` 来查看镜像默认Clusterfile配置：

```shell script
sealer inspect registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest -c
```

## 使用Clusterfile拉起一个k8s集群

使用已经提供好的官方基础镜像(sealer-io/kubernetes:v1.19.8)就可以快速拉起一个k8s集群。

场景1. 往已经存在的服务器上去安装，provider类型为BAREMETAL

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
    # ssh的登录密码，如果使用的密钥登录则无需设置
    passwd:
    # ssh的私钥文件绝对路径，例如/root/.ssh/id_rsa
    pk: xxx
    # ssh的私钥文件密码，如果没有的话就设置为""
    pkPasswd: xxx
    # ssh登录用户
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2

  masters:
    ipList:
     - 172.20.125.234
     - 172.20.126.5
     - 172.20.126.6
  nodes:
    ipList:
     - 172.20.126.8
     - 172.20.126.9
     - 172.20.126.10
```

```
[root@iZm5e42unzb79kod55hehvZ ~]# sealer apply -f Clusterfile
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                      STATUS   ROLES    AGE   VERSION
izm5e42unzb79kod55hehvz   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r7z   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r8z   Ready    master   18h   v1.16.9
izm5ehdjw3kru84f0kq7r9z   Ready    <none>   18h   v1.16.9
izm5ehdjw3kru84f0kq7raz   Ready    <none>   18h   v1.16.9
izm5ehdjw3kru84f0kq7rbz   Ready    <none>   18h   v1.16.9
```

>kubernetes:v1.19.8镜像默认使用calico镜像，服务器网卡名称需符合默认匹配规则`interface: "eth.*|en.*"`
可使用自定义[custom-calico.yaml](https://docs.projectcalico.org/reference/installation/api#operator.tigera.io/v1.Installation) 来[重写默认calico文件](../applications/calico/README.md)

场景2. 自动申请阿里云服务器进行安装, provider: ALI_CLOUD
Clusterfile:

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  provider: ALI_CLOUD
  ssh:
    # ssh的登录密码，如果使用的密钥登录则无需设置
    passwd:
    # ssh的私钥文件绝对路径，例如/root/.ssh/id_rsa
    pk: xxx
    # ssh的私钥文件密码，如果没有的话就设置为""
    pkPasswd: xxx
    # ssh登录用户
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2

  masters:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
```

```
# 准备好阿里云的ak sk
[root@iZm5e42unzb79kod55hehvZ ~]# ACCESSKEYID=xxxxxxx ACCESSKEYSECRET=xxxxxxx sealer apply -f Clusterfile
```

> 释放集群

基础设置的一些源信息会被写入到Clusterfile中，存储在 /root/.sealer/[cluster-name]/Clusterfile中, 所以可以这样释放集群：

```
sealer delete -f /root/.sealer/my-cluster/Clusterfile
或
sealer delete --all
```

## 制作一个自定义的集群镜像, 这里以制作一个dashboard镜像为例

新建一个dashboard目录,创建一个文件Kubefile内容为:

```
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```

```
[root@iZm5e42unzb79kod55hehvZ dashboard]# export ACCESSKEYID=xxxxxxx
[root@iZm5e42unzb79kod55hehvZ dashboard]# export ACCESSKEYSECRET=xxxxxxx
[root@iZm5e42unzb79kod55hehvZ dashboard]# sealer build -f Kubefile -t my-kuberentes-cluster-with-dashboard:latest .
```

创建一个带有dashboard的自定义集群, 操作同上，替换掉Clusterfile中的image字段即可：

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: my-kuberentes-cluster-with-dashboard:latest
  provider: ALI_CLOUD
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
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
```

```
# 准备好阿里云的ak sk
[root@iZm5e42unzb79kod55hehvZ ~]# ACCESSKEYID=xxxxxxx ACCESSKEYSECRET=xxxxxxx sealer apply -f Clusterfile
```

把制作好的集群镜像推送到镜像仓库：

```
sealer tag my-kuberentes-cluster-with-dashboard:latest registry.cn-qingdao.aliyuncs.com/sealer-io/my-kuberentes-cluster-with-dashboard:latest
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/my-kuberentes-cluster-with-dashboard:latest
```

就可以把镜像复用给别人进行使用
