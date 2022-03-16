# Sealer

## 什么是sealer

sealer[ˈsiːlər]是一款分布式应用打包交付运行的解决方案，通过把分布式应用及其数据库中间件等依赖一起打包以解决应用整个集群整体交付问题。
sealer构建出来的产物我们称之为"集群镜像", 集群镜像里内嵌了一个kubernetes, 解决了分布式应用的交付一致性问题。
集群镜像可以push到registry中共享给其他用户使用，也可以在官方仓库中找到非常通用的分布式软件直接使用。

Docker可以把一个操作系统的rootfs+应用 build成一个容器镜像，sealer把kubernetes看成操作系统，在这个更高的抽象纬度上做出来的镜像就是集群镜像。
实现整个集群的Build Share Run !!!

有了集群镜像用户实践云原生生态技术将变得极其简单，如：

### 安装一个kubernetes集群

sealer可以通过一条命令安装一个kubernetes集群，仅需要提供IP列表和ssh访问密码.

```shell script
# 安装sealer
wget https://github.com/alibaba/sealer/releases/download/v0.7.1/sealer-v0.7.1-linux-amd64.tar.gz && \
tar zxvf sealer-v0.7.1-linux-amd64.tar.gz && mv sealer /usr/bin
# 安装kubernetes集群
sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 --nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                      STATUS   ROLES    AGE   VERSION
izm5e42unzb79kod55hehvz   Ready    master   18h   v1.19.8
izm5ehdjw3kru84f0kq7r7z   Ready    master   18h   v1.19.8
izm5ehdjw3kru84f0kq7r8z   Ready    master   18h   v1.19.8
izm5ehdjw3kru84f0kq7r9z   Ready    <none>   18h   v1.19.8
izm5ehdjw3kru84f0kq7raz   Ready    <none>   18h   v1.19.8
izm5ehdjw3kru84f0kq7rbz   Ready    <none>   18h   v1.19.8
```

参数 | 含义 | 示例
---|---|---
kubernetes:v1.19.8| 集群镜像名称，可以直接使用官方已经制作好的镜像| kubernetes:v1.19.8
masters| master地址列表，支持单机和master集群| --masters 192.168.0.2,192.168.0.3,192.168.0.4
nodes| node地址列表，可以为空|--nodes 192.168.0.5,192.168.0.6,192.168.0.7
passwd| ssh访问密码, 更多配置如端口号或者每台主机不一致可以使用Clusterfile，见详细文档| --passwd xxx

### 增删节点

```shell script
# 增加master节点
sealer join --masters 192.168.0.2
# 增加node节点
sealer join --nodes 192.168.0.3
# 删除master节点
sealer delete --masters 192.168.0.2
# 删除node节点
sealer delete --nodes 192.168.0.3
```

### 释放集群

基础设置的一些源信息会被写入到Clusterfile中，存储在 /root/.sealer/[cluster-name]/Clusterfile中, 所以可以这样释放集群：

```
sealer delete -f /root/.sealer/my-cluster/Clusterfile
或
sealer delete --all
```

### 构建一个包含dashboard的自定义集群镜像

经常会有这样的需求，就是有些用户需要集群带上dashboard, 有些人需要calico有些有需要flannel，那如何让用户自定义自己想要的集群？
Sealer最出色的地方是可以非常方便的让用户自定义一个集群的镜像，通过像Dockerfile一样的文件来描述和build：

这里以构建一个包含dashboard的集群镜像为例，recommended.yaml就是包含dashboard的deployment service等yaml文件
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

把制作好的集群镜像推送到镜像仓库，集群镜像仓库兼容docker镜像仓库标准，可以把集群镜像推送到docker hub、阿里ACR、或者Harbor中

```shell script
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

一键运行包含dashboard的集群镜像：

```shell script
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest \
    --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
    --nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
```

## 使用场景&特性

- [x] 极其简单的方式在生产环境中或者离线环境中安装kubernetes、以及kubernetes生态中其它软件
- [x] 通过Kubefile可以非常简单的自定义kubernetes集群镜像对集群和应用进行打包，并可以提交到仓库中进行分享
- [x] 强大的生命周期管理能力，以难以想象的简单的方式去做如集群升级，集群备份恢复，节点扩缩等操作
- [x] 速度极快3min以内完成集群安装
- [x] 支持ARM x86, v1.20以上版本支持containerd，几乎兼容所有支持systemd的linux操作系统
- [x] 不依赖ansible haproxy keepalived, 高可用通过ipvs实现，占用资源少，稳定可靠
- [x] 官方仓库中有非常多的生态软件镜像可以直接使用，包含所有依赖，一键安装
- [x] 自动对接公有云基础设施
- [x] 强大的配置管理能力，非常方便灵活的调整kubeadm安装参数，或者集群镜像内部的业务组件参数
- [x] 强大的插件能力，支持在安装的各各阶段自定义自己的操作，如修改主机名与时间同步等操作，支持out of tree插件开发