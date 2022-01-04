# docker镜像及helm chart包保存

save模块的位置在image/save目录下，其作用是拉取其他仓库中的docker镜像或者helm chart包，并保存到本地的文件系统中。

## 使用示例

### Helm chart包
 
> workflow: 用helm push 推送chart包到registry中->使用sealer build将chart包存储到集群镜像中->用build后的集群镜像建立集群，使用helm pull拉取仓库中的chart包

#### Step1: helm push 推送chart包到registry中

例如，本地存在两个已经构建好的chart包：mysql-8.8.19-tgz和rabbitmq-8.24.13.tgz。首先用registry:2.7.1镜像启动docker容器监听5000端口，然后设置环境变量`HELM_EXPERIMENTAL_OCI=1`，接着执行`helm push mysql-8.8.19-tgz oci://localhost:5000/helm-charts`,`helm push rabbitmq-8.24.13.tgz oci://localhost:5000/helm-charts`。

#### Step2: 使用sealer build将chart包存储到集群镜像中

在一个空目录下创建两个文件，一个是`Kubefile`文件，一个是`imageList`文件。`Kubefile`文件内容如下:

```
FROM kubernetes-clusterv2:v1.19.8
COPY imageList manifests
```

`imageList`文件内容如下:

```
localhost:5000/helm-charts/mysql:8.8.19
localhost:5000/helm-charts/rabbitmq:8.24.13
```

在该目录下执行`sealer build -t test:1 .`

#### Step3: 建立集群并用helm pull拉取仓库中的chart包

需要一台额外的机器来创建k8s集群，其ip地址用`x.x.x.x`表示。执行`sealer run test:1 -m x.x.x.x -p x`创建集群。集群创建成功后，进入ip地址为`x.x.x.x`的机器，安装helm，设置环境变量`HELM_EXPERIMENTAL_OCI=1`。然后可以通过执行`helm pull oci://sea.hub:5000/helm-charts/mysql --version 8.8.19`和`helm pull oci://sea.hub:5000/helm-charts/rabbitmq --version 8.24.13`来拉取相应的chart包。

### Docker 镜像

> workflow: 使用sealer build将docker镜像存储到集群镜像中->用build后的集群镜像建立集群，使用docker pull拉取仓库中的docker镜像

#### Step1: 使用sealer build将docker镜像存储到集群镜像中

在一个空目录下创建两个文件，一个是`Kubefile`文件，一个是`imageList`文件。`Kubefile`文件内容如下:

```
FROM kubernetes-clusterv2:v1.19.8
COPY imageList manifests
```

`imageList`文件内容如下:

```
mysql:latest
ubuntu:18.04
```

如果要拉取私有仓库的镜像，请先执行`docker login` 登录。在该目录下执行`sealer build -t test:1 .`

#### Step2: 建立集群并用docker pull拉取仓库中的docker镜像

需要一台额外的机器来创建k8s集群，其ip地址用`x.x.x.x`表示。执行`sealer run test:1 -m x.x.x.x -p x`创建集群。集群创建成功后，进入ip地址为`x.x.x.x`的机器，执行`docker pull sea.hub:5000/mysql:latest`和`docker pull sea.hub:5000/ubuntu:18.04`来拉取相应的docker镜像。


## 模块工作流程

1. 解析镜像名称，获得三个重要信息：域名（registry），仓库名（repository），镜像名（image），把相同域名的镜像放在一个切片中，以便后续一起处理。
2. 根据域名建立与相应的registry的连接。然后就能获取到repository service, manifest service, tag service, blob service. 通过这些service来拉取和保存镜像数据。
3. 获取manifest列表，该列表包含所有架构的镜像。每一个镜像都有一个唯一的digest字段。通过digest字段可以获取到该镜像的manifest数据。
4. 获取blob数据。每一个镜像都包含多个blob，每一个blob也有一个唯一的digest字段。通过步骤3中的manifest可以获取到所有的blob的digest字段，然后再通过blob的digest字段能够获取到所有的blob的数据。
5. 把一个镜像的manifest和blob都拉取下来之后，保存到本地的文件系统下。这样，启动一个私有registry时，只需要挂载相应的目录，私有registry中就包含了目录下的所有镜像。

## 对外暴露接口

save模块对外只暴露一个接口，如下：

```
type ImageSave interface {
	SaveImages(images []string, dir, arch string) error
}
```

`SaveImages`函数接收三个参数，返回一个error信息。三个参数分别代表着：

1. images：要保存的所有的镜像名称，类型是一个字符串slice
2. dir：保存镜像的目的路径，类型是string
3. arch：要保存的镜像是基于什么处理器架构的，类型是string

save模块定义了一个DefaultImageSaver对象，该对象实现了ImageSave接口，同时还提供了该对象的New函数。

```
type DefaultImageSaver struct {
	ctx            context.Context
	domainToImages map[string][]Named
}

func NewImageSaver(ctx context.Context) ImageSave {
	if ctx == nil {
		ctx = context.Background()
	}
	return &DefaultImageSaver{
		ctx:            ctx,
		domainToImages: make(map[string][]Named),
	}
}
```

`DefaultImageSaver`对象有两个字段，一个是`context.Context`类型，表示代码执行的上下文环境。另一个是从`string`到`[]Named`的映射，`string`表示域名，`[]Named`是镜像名称的切片，该映射其实是从域名到该域名下的所有镜像名称的映射。

`NewImageSaver`函数接收一个`context.Context`类型的参数，若该参数为空，则调用`context.Background()`创建一个`context.Context`类型的对象用来对`DefaultImageSaver`进行初始化。

`ImageSave`接口的使用示例如下：

```
images := []string{"ubuntu", "ubuntu:18.04", "registry.aliyuncs.com/google_containers/coredns:1.6.5", "fanux/lvscare"}
is := NewImageSaver(context.Background())
err := is.SaveImages(images, "/var/lib/registry", "amd64")
if err != nil {
	panic(err)
}
```