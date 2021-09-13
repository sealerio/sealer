# 镜像构建

> 就像使用Dockerfile来构建容器镜像一样， 我们可以通过Kubefile来定义一个sealer的集群镜像。我们可以使用和Dockerfile一样的指令来定义一个可离线部署的交付镜像。


## Kubefile 样例

For example:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this CloudImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

## Kubefile 指令说明

### FROM 指令

FROM: 引用一个基础镜像，并且Kubefile中第一条指令必须是FROM指令。若基础镜像为私有仓库镜像，则需要仓库认证信息，另外 sealer社区提供了官方的基础镜像可供使用。

> 命令格式：FROM {your base image name}

使用样例：

例如上面示例中,使用sealer 社区提供的`kubernetes:v1.19.9`作为基础镜像。

`FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9`

### COPY 指令

COPY: 复制构建上下文中的文件或者目录到rootfs中。

集群镜像文件结构均基于[rootfs结构](../../docs/api/cloudrootfs.md),默认的目标路径即为rootfs，且当指定的目标目录不存在时会自动创建。

如需要复制系统命令，例如复制二进制文件到操作系统的$PATH，则需要复制到rootfs中的bin目录，该二进制文件会在镜像构建和启动时，自动加载到系统$PATH中。

> 命令格式：COPY {src dest}

使用样例：

复制mysql.yaml到rootfs目录中。

`COPY mysql.yaml .`

复制可执行文件到系统$PATH中。

`COPY helm ./bin`

### RUN 指令

RUN: 使用系统shell执行构建命令，可接受多个命令参数，且构建时会将命令执行产物结果保存在镜像中。若系统命令不存在则会构建失败,则需要提前执行COPY指令，将命令复制到镜像中。

> 命令格式：RUN {command args ...}

使用样例：

例如上面示例中,使用wget 命令下载一个kubernetes的dashboard。

`RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml`

### CMD 指令

CMD: 与RUN指令格式类似，使用系统shell执行构建命令。但CMD指令会在镜像启动的时候执行，一般用于启动和配置集群使用。另外与Dockerfile中CMD指令不同，一个kubefile中可以有多个CMD指令。

> 命令格式：CMD {command args ...}

使用样例：

例如上面示例中,使用 kubectl 命令安装一个kubernetes的dashboard。

`CMD kubectl apply -f recommended.yaml`

## build类型

> 针对不同的业务需求场景，sealer build 目前支持3种构建方式。

### 1. cloud build

> 默认的build类型。基于云服务（目前仅支持阿里云， 欢迎贡献其他云厂商的Provider），自动化创建ecs并部署kubernetes集群并构建镜像，cloud build 是兼容性最好的构建方式， 基本可以100%的满足构建需求。缺点是需要创建按量计费的云主机会产生一定的成本。如果您要交付的环境涉及例如分布式存储这样的底层资源，建议使用此方式来进行构建。

```shell
sealer build -t my-cluster:v1.19.9 .
```

### 2. container build

> 与cloud build 原理类似，通过启动多个docker container作为kubernetes节点（模拟cloud模式的ECS）,从而启动一个kubernetes集群的方式来进行构建，可以消耗很少量的资源完成集群构建，缺点是不能很好的支持对底层资源依赖的场景。可以使用`-b container` 参数来指定build 类型 为container build 。

```shell
sealer build -b container -t my-cluster:v1.19.9 .
```

### 3. lite build

最轻量的构建方式， 原理是通过解析helm chart、提交镜像清单、解析manifest下的资源定义获取镜像清单并缓存，
配合Kubefile的定义，实现不用拉起kubernetes集群的轻量化构建，此种方式优点是资源消耗最低，有一台能够跑sealer的主机即可进行构建。缺点是无法覆盖一些场景，
例如无法获取通过operator部署的镜像，一些通过专有的管理工具进行交付的业务也无法解析获取到对应的镜像，lite build适用于已知镜像清单， 或者没有特殊的资源需求的场景。

Kubefile 示例：

```shell
FROM kubernetes:v1.19.9
COPY imageList manifests
COPY apollo charts
RUN helm install charts/apollo
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

> 注意： 在lite build的场景下，因为build过程不会拉起集群，类似kubectl apply和helm install并不会实际执行成功， 但是会作为镜像的一层在交付集群的时候执行。

如上示例，lite构建会从如下三个位置解析会获取镜像清单，并将镜像缓存至registry：

* manifests/imageList: 内容就是镜像的清单，一行一个镜像地址。如果这个文件存在，则逐行提取镜像。imageList的文件名必须固定，不可更改，且必须放在manifests下。
* manifests 目录下的yaml文件: lite build将解析manifests目录下的所有yaml文件并从中提取镜像。
* charts 目录: helm chart应放置此目录下， lite build将通过helm引擎从helm chart中解析镜像地址。

lite build 操作示例，使用`-b lite` 参数来指定build 类型为 lite build。 假设Kubefile在当前目录下：

```shell
sealer build -b lite -t my-cluster:v1.19.9 .
```

构建完成将生成镜像：my-cluster:v1.19.9

## 私有仓库认证

在构建过程中，会存在使用私有仓库需要认证的场景， 在这个场景下， 进行镜像缓存时需要依赖docker的认证。可以在执行build操作前通过以下指令先进行login操作：

```shell
sealer login registry.com -u username -p password
```

另一个依赖场景，在交付完成后的，kubernetes node通过sealer内置的registry 代理到私有仓库且私有仓库需要认证时，可以通过自定义registry config来配置，sealer
优化和扩展了registry，使其可以同时支持多域名，多私有仓库的代理缓存。配置可参考: [registry配置文档](../user-guide/docker-image-cache.md)

可以通过定义Kubefile来自定义registry配置:

```shell
FROM kubernetes:v1.19.9
COPY registry_config.yaml etc/registry_config.yaml
```
