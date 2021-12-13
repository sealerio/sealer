# docker镜像保存

save模块的位置在image/save目录下，其作用是拉取其他仓库中的docker镜像，并保存到本地的文件系统中。

## 整体工作流程

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

TODO：
1.目前，拉取到的manifest列表，代码只支持对schema2版本manifest的处理，没有对其他版本的处理。
2.我觉得目前的镜像拉取时并发做的不够好，还可以继续优化。
3.对于如何选择不同架构的镜像，目前只是进行字符串匹配，可以优化。（参考docker所采用的最佳匹配）