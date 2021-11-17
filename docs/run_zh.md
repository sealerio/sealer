# sealer run

> 使用已经build好的镜像，一条命令快速拉起集群，原理同apply类似，run命令使用更加方便快捷，使用一条命令即可部署整个集群。

执行命令解析：

```yaml
$ sealer run kubernetes:v1.19.8 # 使用镜像中Clusterfile配置启动集群
或
$ sealer run kubernetes:v1.19.8 -m 3 -n 3 -p [自定义密码] --provider ALI_CLOUD #CLOUD模式 自定义节点数量
或
$ sealer run kubernetes:v1.19.8 -m 3 -n 3 -p [自定义密码] --provider CONTAINER #CONTAINER模式 自定义节点数量
或
$ sealer run kubernetes:v1.19.8 -m 192.168.56.113,192.168.56.114 -n 192.168.56.115,192.168.56.116 -p xxx  #BAREMETAL模式 使用已有机器

  -m, --masters : master节点数量
  -n, --nodes   : node 节点数量
  -u, --user    : 机器用户名，默认为root用户
  -p, --passwd  : 为[CLOUD | CONTAINER]模式自定义密码，BAREMETAL模式为已有机器密码
  --provider    : 设置启动模式：[ALI_CLOUD | CONTAINER ｜BAREMETAL]，默认为BAREMETAL
  --pk          : BAREMETAL模式设置私钥文件，默认为$HOME/.ssh/id_rsa文件
  --pk-passwd   : BAREMETAL模式设置私钥密码
  --podcidr     : 设置默认pod CIDR （需要与calico中自定义配置的podCIDR一致）
  --svccidr     : 设置默认service CIDR
```

查看镜像默认启动配置：

```shell
sealer inspect kubernetes:v1.19.8 -c #查看kubernetes:v1.19.8镜像中的Clusterfile
```

## 设置镜像自定义启动Clusterfile

> ### build镜像时默认设置Clusterfile为基础镜像中Clusterfile，也可在build时使用自定义Clusterfile：

例：

```shell
$ mkdir build && cd build #创建build上下文
$ vi Clusterfile # 创建自定义Clusterfile
$ vi Kubefile # 创建Kubefile
#Kubefile
FROM kubernetes:v1.19.8
COPY Clusterfile . #该步骤将自动识别并设置自定义Clusterfile为默认启动Clusterfile
```

```shell
sealer build -t my-kubernetes:v1.19.8 . #执行成功即可生成含有自定义Clusterfile的镜像

sealer inspect my-kubernetes:v1.19.8 -c #查看集群镜像默认Clusterfile
```