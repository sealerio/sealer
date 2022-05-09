# 集群镜像 build

## Build 类型层

该模块主要通过对不同构建类型的区分，完成对集群镜像构建前期环境的准备，例如启动K8S 集群，启动sealer registry,以及拉取用于构建的镜像，从而分别实现Build接口，完成对集群镜像的构建。

### 类型介绍

目前sealer默认使用lite build方式，详细使用文档见[build 使用文档](http://sealer.cool/docs/getting-started/build-cloudimage.html) 。

#### lite build

该种构建模式，根据Kubefile的定义，实现不用拉起kubernetes集群的轻量化构建。用于已知镜像清单，或者没有特殊的资源需求的场景。

核心功能介绍

1. 基础集群镜像的拉取和挂载。
2. 本地缓存registry的启动和挂载。
3. 根据kubefile执行构建指令,收集相关产物。
4. 收集缓存的容器镜像。
5. 清理环境，本地缓存registry的回收。

### 接口定义

`Build(name string, context string, kubefileName string) error`

参数解释

```yaml
name: 新构建的集群镜像名字，不支持大写和特殊字符。
context: 用于构建集群镜像的上下文，默认为当前目录。
kubefilename: kubefile 用于描述集群镜像的构建过程，和Dockerfile类似。该参数为用户自定义的kubefile文件路径。
error: 当执行构建时候，如有发生错误，则会返回该错误。
```

## Build 镜像层

集群构建流程的核心的实现，主要通过解析kubefile，初始化镜像模块接口，根据集群镜像的layer层定义，执行对应的指令内容和收集对应的产物，从而生成完整的集群镜像。

核心功能介绍

1. kubefile 文件的解析。
2. 镜像模块接口初始化，获取当前镜像的基础镜像信息和当前镜像层。
3. 根据文件目录生成新的镜像层
4. 构建指令的执行,以及对应的layer内容的预处理，对应的指令产物收集。
5. 新集群镜像的生成和提供对文件系统的存储功能。

### 接口定义

```shell
SaveBuildImage(name string) error
ExecBuild(ctx Context) error
Cleanup() error
```

参数解释

Context 结构体

```yaml
BuildContext: 构建的上下文。
BuildType: 构建的类型。
UseCache: 标志位，用于检测是否使用缓存。
```

### 构建流程介绍

Kubefile 举例如下：

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
COPY recommended.yml manifests
COPY imageList manifests
COPY traefik ./charts
COPY bin/helm /bin
RUN helm --help
CMD kubectl apply -f manifests/recommended.yml
CMD helm install mytest charts/traefik
```

1. 根据Kubefile 初始化对应的BuildImage `NewBuildImage(kubefileName string)`，得到一个BuildImage结构体。
2. 根据Kubefile的内容，判断是否需要获取基础镜像，是否需要启动缓存registry，并且初始化镜像模块的接口。
3. 按照Kubefile对应的每一层内容，初始化不同的build指令，并且执行该指令，收集对应的产物。
4. 根据该层指令内容触发对应的layer 层的handler。例如 `COPY imageList manifests` 则会解析imageList中的镜像，并且缓存起来。
5. 待Kubefile对应的每一层内容执行完成后，根据是否启动缓存registry，判断是否需要收集对应的docker 镜像。
6. 待所有产物收集完毕，调用镜像模块接口将新构建的镜像保存到文件系统。
7. 环境清理，例如构建时候产生的临时文件，缓存registry的回收等。

## Build 指令层

该模块主要完成对不同的构建指令的生成和执行，包括 `copy`,`cmd`,`run` 等具体指令的实现，以及集群镜像的层级挂载，集群镜像的缓存判断，指令层的对应layer id生成，对应layer层产物的存储等等。

### 指令介绍

copy 指令

* 实现构建上下文的copy功能。
* 根据指令内容，实现对应的layer层功能。
* 根据指令产物，计算对应指令的layer id。

cmd 指令

* 挂载基础镜像和该指令之前所以指令层产物。
* 根据指令内容，实现对应的layer层功能。
* 根据挂载的指令层产物，实现对应的系统命令的调用。
* 一个kubefile，可以有多个cmd指令，但cmd指令没有产物，即不会计算对应的指令layer id。

run 指令

* 实现逻辑同cmd指令一致，但是会根据指令产物计算对应指令的layer id。

### 指令缓存

#### 概念介绍

* layer id : 根据每一层指令运行的产物，根据时间，文件内容，文件权限等计算出该层指令独一无二的ID。
* cache id : 根据context 中文件的内容，时间权限等计算出该层指令独一无二的ID，如没有命中，则会在该指令层，生成对应的cacheID文件，内容为cache id。
* cache layer: 包括cache id和layer value，layer type的新结构体。
* parent id : 不包括当前层之前的所有指令的chain id计算结果。
* chain id : 根据parent id 和cache layer计算得出，用与匹配文件系统已经缓存的集群镜像。

#### 缓存计算方式

默认的parent id 起始值为""，系统会在初始化时候，按照以下生成逻辑，遍历当前文件系统存在的所有集群镜像。格式为map,key 是当前层的chain id，value为当前层对应的cache layer.

Copy指令 chain id 生成举例

```yaml
chain id = ChainID（"parent id"+"cache id"+"COPY:recommended.yaml manifests"）
```

非Copy指令 chain id 生成举例

非Copy指令没有cache id，则计算时候为""。

```yaml
chain id = ChainID（"parent id" +""+"CMD:kubectl apply -f etc/tigera-operator.yaml"）
```

#### 缓存命中逻辑

1. 根据cache 标志位判断是否需要进行缓存。若是则走以下流程，若否则走常规计算流程。
2. 计算context 中文件的对应的cache id,并且和当前对应的parent id进行计算，得到当前指令层的chain id。
3. 根据 chain id 和系统中存在的所有缓存map，进行比较。
4. 若命中，返回当前的layer id ，cache 标志设置为true，并将当前parent id替换为最新的chain id。
5. 若没有命中，返回当前的layer id ，cache 标志设置为false，当前parent id不发生变化，并将该层对应的cache id 写入文件系统，用于下次缓存计算使用。

### 接口定义

```shell
Exec(ctx ExecContext) (out Out, err error)
```

参数解释

ExecContext 结构体

```yaml
BuildContext: 构建的上下文。
BuildType: 构建的类型。
ContinueCache: 标志位，用于检测是否继续使用缓存，会在每一层layer执行的时候修改。
ParentID cache.ChainID: 缓存chain id，起始值为 ""。会在每一层layer执行的时候修改为包括当前layer的chain id。
CacheSvc cache.Service: 缓存layer生成接口，根据当前layer和cache id,生成对应的缓存layer。用于和ParentID一起，生成当前的chain id。
Prober image.Prober: 缓存探测接口，用于判断对应的chain id 是否和系统中存在的缓存map命中。
LayerStore store.LayerStore: 构建产物存储接口，会根据当前指令执行的结果，计算对应的layerid，并将产物复制到对应的文件系统路径中。
```

Out 结构体

```yaml
LayerID digest.Digest: 当前指令执行的产物hash,独一无二，将会用于当前layer的存储。
ParentID cache.ChainID: 缓存chain id，与ExecContext 中ParentID一致，用于下一层指令的缓存判断。
ContinueCache bool: 标志位，用于判断下一层指令是否继续使用缓存。
```

## Build layer层

该模块主要完成对layer层内容的解析以及对应的处理函数的初始化。

### 功能介绍

#### 解析 imageList

使用方式

如果对应的Kubefile中有以下字段，则会触发解析 imageList功能。其中COPY的 src 名字为`imageList`,dest 为`manifests`。

```shell
COPY imageList manifests
```

功能介绍

1. imageList 解析，将对应`imageList`中的内容逐行获取。
2. 镜像拉取，将解析到的对应docker镜像，使用docker client拉取到本地。

#### 解析helm charts

使用方式

如果对应的Kubefile中有以下字段，则会触发解析helm charts功能。其中COPY的 src 名字为charts 包,dest 为`charts`。

```shell
COPY traefik charts
```

功能介绍

1. charts包 解析，将对应 charts 包 中的内容使用helm引擎获取docker 镜像列表。
2. 镜像拉取，将解析到的对应docker镜像，使用docker client拉取到本地。

#### 解析yaml 文件

使用方式

如果对应的Kubefile中有以下字段，则会触发解析yaml 文件缓存镜像功能。其中COPY的 src 名字为 yaml文件格式为"yaml"或者"yml",dest 为`manifests`。

```shell
COPY recommended.yml manifests
```

功能介绍

1. yaml文件解析，读取对应yaml文件中的内容，解析`image`字段获取docker镜像列表。
2. 镜像拉取，将解析到的对应docker镜像，使用docker client拉取到本地。

### 接口定义

```shell
LayerValueHandler(buildContext string, layer v1.Layer) error
```

参数解释

```yaml
buildContext: 用于构建集群镜像的上下文，默认为当前目录。
layer: 构建阶段，对应的集群镜像层的内容，包括layer value,layer type。
error: 当执行构建时候，如有发生错误，则会返回该错误。
```

### docker镜像缓存
如果使用的是sealer定制的docker，则docker镜像缓存到私有仓库只需要一步：从官方仓库pull镜像。因为sealer把私有仓库设置为代理，所有镜像都会先pull到私有仓库，然后才会再从私有仓库pull到本地。

如果使用的是原生的docker，则docker镜像缓存到私有仓库需要两步：首先，从官方仓库pull镜像。然后，将pull到本地的镜像再push到私有仓库中。

在每一个镜像的拉取操作之前加入一个判断docker引擎版本的步骤来进行区分。具体实现方式是：调用docker的sdk函数，得到docker引擎的版本信息，若docker引擎的版本中包含`sealer`字符串，则标示是sealer定制的docker，只进行pull操作。若docker引擎的版本中，不包含`sealer`字符串，则pull和push操作都会执行。
