# build 产物

1. docker 镜像。
2. 构建过程。
3. 构建过程文件。

其中，构建过程和构建过程产出的文件，均可以在模拟器中实现。 只有docker 镜像涉及系统硬件架构，具体的和docker deamon 相关。

# 方案一

启动arm 容器，启动registry,使用sealer-docker，通过挂载本地docker.sock，实现docker镜像获取。

## 配置系统内核兼容arm指令

`docker run --rm --privileged multiarch/qemu-user-static:register`
可以看见对应的arm 二进制文件的 interpreter。
`ll /proc/sys/fs/binfmt_misc/`

## 启动arm 容器

挂载本地docker.sock， 挂载本地sealer 目录。

`docker run -itd --privileged --security-opt seccomp=unconfined --security-opt apparmor=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /var/run/docker.sock:/var/run/docker.sock --volume /var/lib/sealer:/var/lib/sealer --device /dev/fuse --detach --tty --restart=on-failure:1 --init=false registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image-arm:latest`

复制 arm sealer 二进制 复制arm docker client 二进制

```shell
docker cp docker {containerID}:/usr/local/bin/docker
docker cp sealer {containerID}:/usr/local/bin/sealer
```

## 启动registry

```shell
mkdir -p /var/lib/sealer/tmp/registry
docker run -d --restart=always --net=host --name sealer-registry -v /var/lib/sealer/tmp/registry:/var/lib/registry registry:2.7.1
```

arm container 中配置/etc/hosts: `{containerIP} sea.hub`

## 测试pull docker

docker pull busybox:latest

curl sea.hub:5000/v2/_catalog

tree /var/lib/sealer/tmp/registry

## 执行 sealer build

进入到arm container 中:

```shell
sealer build -f Kubefile -t myimage:v1 -m lite .
```

# 方案二

启动arm 容器，完成sealer build 的产物和过程收集，docker镜像通过集成docker registry 源码实现镜像的拉取。这样只需要启动arm容器，与docker 无关。

## 配置系统内核兼容arm指令

`docker run --rm --privileged multiarch/qemu-user-static:register`
可以看见对应的arm 二进制文件的 interpreter。
`ll /proc/sys/fs/binfmt_misc/`

## 启动arm 容器

挂载本地sealer目录。无需挂载本地docker.sock。

`docker run -itd --privileged --security-opt seccomp=unconfined --security-opt apparmor=unconfined --tmpfs /tmp --tmpfs /run --volume /var --volume /var/lib/sealer:/var/lib/sealer --device /dev/fuse --detach --tty --restart=on-failure:1 --init=false registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image-arm:latest`

复制 arm sealer 二进制。

```shell
docker cp sealer {containerID}:/usr/local/bin/sealer
```

## 执行 sealer build

构建过程中的docker 镜像获取都由集成的registry源码实现，保存到自定义的本地路径，最终会被计算加载到对应集群镜像的一层layer。

进入到arm container 中:

```shell
sealer build -f Kubefile -t myimage:v1 -m lite .
```

# 方案三

不启动arm容器，本地也无需额外配置docker环境。只需要配置qemu 模拟器以及对应的环境就可以。

## 配置系统内核兼容arm指令

`docker run --rm --privileged multiarch/qemu-user-static:register`
可以看见对应的arm 二进制文件的 interpreter。
`ll /proc/sys/fs/binfmt_misc/`

## 配置qemu 模拟器

下载qemu user static 模拟器到本地，这样对应的arm二进制都可以被该模拟器运行。

```shell
curl -L -o qemu-arm-static-v2.11.1.tar.gz https://github.com/multiarch/qemu-user-static/releases/download/v6.1.0-8/qemu-aarch64-static.tar.gz
tar xzf qemu-arm-static-v2.11.1.tar.gz
cp qemu-arm-static /usr/bin/
```

举例运行arm 版本的二进制docker文件

```shell
/usr/bin/qemu-arm-static docker ps
```

## build流程中接管所有build 指令

## 对应产物收集

* 构建过程产物收集
* docker arm 镜像收集