# 集群镜像apply
>使用已经build好的镜像来快速拉起集群，集群的信息通过Clusterfile文件来定义。

## Clusterfile 定义

```shell
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  #集群名称 可自定义
  name: my-cluster
spec:
  #通过集群镜像拉起集群
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  #provider: 使用阿里云服务器设置为ALI_CLOUD，使用docker并以容器的方式创建集群设置为CONTAINER，使用自有服务器设置为BAREMETAL
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
    # CIDR需要与cni中定义CIDR一致，如需修改CIDR且使用官方含calico镜像则需要在Clusterfile中添加自定义config来替换自定义calico配置
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  # 添加DNS名称或者IP地址到apiServer:
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
  # ALI_CLOUD模式通过count数量拉起ecs服务器并创建集群, 如果为BAREMETAL模式，使用ipList.
  masters:
    #cpu,memory,systemDisk,dataDisks为Cloud模式设置集群配置，BAREMETAL模式无需关心
    cpu: 4
    memory: 4
    #ALI_CLOUD模式通过count数量进行基础设置的拉起与删除及节点添加删除，无需修改ipList
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  #BAREMETAL通过ipList进行节点添加与删除
  # ipList:
  #  - 192.168.0.2
  #  - 192.168.0.3
  #  - 192.168.0.4
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  # ipList:
  #  - 192.168.0.5
  #  - 192.168.0.6
  #  - 192.168.0.7
```

>如果使用ARM64机器，使用registry.cn-beijing.aliyuncs.com/sealer-io/kubernetes-arm64:v1.19.7作为集群镜像。

## 创建集群

```shell
执行命令:
sealer apply -f Clusterfile
```

**sealer 通过自定义Clusterfile文件获取集群信息，并通过集群镜像创建集群。**

**执行成功后会生成/root/.sealer/[集群名称]/Clusterfile用来存储Cluster信息供sealer使用。**

sealer官方kubernetes:v1.19.9镜像默认使用calico作为网络插件，可以以`kubernetes:v1.19.9-alpine`镜像（不含cni）为基础镜像自定义制作含有其他cni的镜像；

- 默认CIDR为100.64.0.0/10；
- 默认使用IPIP模式（azure云服务器无法使用[IPIP模式](https://docs.projectcalico.org/reference/public-cloud/azure), 需修改calico配置为VXLAN）；
- 默认使用`interface: "eth.*|en.*"`进行网络接口匹配；
- 以上配置可使用Config功能来[自定义calico配置](../applications/calico/README.md)替换etc/custom-resources.yaml作为执行文件。
- [Config](./design/global-config_zh.md)与[Plugin](./design/plugin_zh.md)使用方法类似，在安装集群时在Clusterfile内容后追加Config与Plugin，可包含多个Config与Plugin，使用`---`隔开。
  **plugin插件功能可以帮助用户做一些之外的事情，比如更改主机名，升级内核，或者添加节点标签等：[Plugin使用文档](./design/plugin_zh.md)**
- provider为ALI_CLOUD模式需要设置阿里云[AK，SK](https://ram.console.aliyun.com/manage/ak) 并通过拉起ecs服务器的形式启动k8s集群，provider为CONTAINER模式需要安装docker并以docker容器的方式创建k8s集群。

## 添加节点

### CLOUD 模式（目前支持阿里云ecs服务器），CONTAINER模式：

```shell
$ sealer join -m 1 -n 1
  -m :    指定添加master节点数量
  -n :    指定添加node节点数量
  -c :    在$HOME/.sealer下的目录不唯一时，指定cluster-name用于读取生成Clusterfile
```

### BAREMETAL 模式：

```shell
sealer join -m 192.168.56.10,192.168.56.11 -n 192.168.56.12,192.168.56.13 #多个IP直接使用`，`号分割

sealer join -m 192.168.56.10-192.168.56.19 -n 192.168.56.20-192.168.56.29 #加入多个连续IP节点
```

## 删除部分节点

### CLOUD 模式，CONTAINER模式：

```shell
$ sealer delete -m 1 -n 1
  -m, --masters    :    指定删除master节点数量
  -n, --nodes      :    指定删除node节点数量
  -f, --Clusterfile:    指定删除Clusterfile中的集群，默认为$HOME/.sealer/[my-cluster]/Clusterfile（通过sealer创建集群后自动生成），
                        如果$HOME/.sealer下存在多个目录，则需要通过该参数指定Clusterfile路径
  --force:  跳过询问直接执行删除
```

### BAREMETAL 模式：

```shell
sealer delete -m 192.168.56.10,192.168.56.11 -n 192.168.56.12,192.168.56.13 #多个IP直接使用`，`号分割

sealer delete -m 192.168.56.10-192.168.56.19 -n 192.168.56.20-192.168.56.29 #删除多个连续IP节点
```

## 删除全部节点

```shell
$ sealer delete -a
或
$ sealer delete --all
或
$ sealer delete -f /root/.sealer/[my-cluster]/Clusterfile
```