# 使用nydus加速文件分发

[Nydus](https://github.com/dragonflyoss/image-service)的按需加载和预取能够极大地提高sealer分发rootfs的性能。

## 分发流程

在服务端，先调用[nydus-image](https://github.com/dragonflyoss/image-service/blob/master/docs/nydus-image.md)将需要分发的目录转换为nydus格式的blob，并创建的同名clientfile目录存放生成的bootstrap文件和nydusd相关配置文件，然后启动[nydusd_http_server](https://github.com/dragonflyoss/image-service/blob/master/contrib/nydus-backend-proxy/README.md)服务。文件分发时，把[nydusd](https://github.com/dragonflyoss/image-service/blob/master/docs/nydusd.md)和相关脚本文件传输到目标节点上，在节点上启动nydusd服务，nydusd会从服务端按需拉取数据并缓存到本地，当所有数据都拉取完后，即使断开nydusdserver也不影响rootfs的使用。

## 快速使用

### 服务端

#### nydusd_http_server start

```bash
sh serverstart.sh
    -i xxx.xxx.xxx.xxx                            #服务端IP
    -d /path/to/rootfsdir1,/path/to/rootfsdir2    #需要分发的目录路径，支持多目录（用于支持sealer的都架构运行），用逗号隔开
```

调用serverstart.sh脚本，输入需要分发的目录路径和服务端IP。
这会调用nydus-image把指定的本地目录转换成blob保存在nydusblobs目录下，创建对应的同名目录（rootfsdir1和rootfsdir2），生成对应的bootstrap文件（rootfsdir1/rootfs.meta）和nydusd配置文件（rootfsdir1/httpserver.json），然后启动nydusdserver服务（nydusd_http_server.service）。

#### nydusd_http_server clean

```bash
sh serverclean.sh
```

调用serverclean.sh，停止并删除nydusdserver服务

### 节点端

#### nydusd start

```bash
sh start.sh /path/to/mount
```

运行nydusd_scp_file/start.sh脚本，输入挂载点路径。
创建并启动nydusd.service，并在挂载点挂载一个overlay文件系统。nydusd会从服务器按需拉取文件并缓存在./cache目录下。

#### nydusd clean

```bash
umount /path/to/mount
sh clean.sh
```

先将挂载点umount，然后调用nydusd_scp_file/start.sh脚本，停止nydusd.service并清理nydusd相关文件和目录。

## nydus指南

### nydusdfile目录结构

```bash
nydusdfile
├── rootfsdir1(clientfile)  # 需要被传输到远端节点的目录，与被分发的本地目录同名，包括nydusd和相应的脚本文件
│   ├── clean.sh            # nydusd clean
│   ├── httpserver.json     # nydusd配置文件，由serverstart.sh脚本生成，配置说明详见[nydusd](https://github.com/dragonflyoss/image-service/blob/master/docs/nydusd.md)
│   ├── nydusd              # nydusd
│   ├── nydusd.service      # systemd service文件，由start.sh脚本生成
│   ├── rootfs.meta         # nydus-image转换目录时生成的bootstrap文件
│   └── start.sh            # nydusd启动脚本
└── serverfile              # nydusd server端文件
    ├── nydus-backend-proxy # nydusd HTTP服务器，将本地目录用作nydusd的blob后端
    ├── nydusblobs          # 存放生成的nydus blobs文件
    ├── nydus-image         # 将目录转换成nydus格式，生成元数据文件bootstrap
    ├── Rocket.toml         # nydus-backend-proxy的配置文件
    ├── serverclean.sh      # nydusd_http_server清理脚本
    └── serverstart.sh      # nydusd_http_server启动脚本
```

### nydus应用使用说明

#### [nydus-image](https://github.com/dragonflyoss/image-service/blob/master/docs/nydus-image.md)

将本地目录转换成nydus格式,生成bootstrap和blob:

- bootstrap：存储目录元数据信息的文件
- blob：存储目录里所有文件数据的文件

```bash
nydus-image create \
  --bootstrap /path/to/bootstrap \      #指定生成的bootstrap文件路径
  --blob /path/to/blob \                #指定生成的blob存放目录
  /path/to/source/dir                   #被转换的本地目录
```

#### [nydus-backend-proxy](https://github.com/dragonflyoss/image-service/blob/master/contrib/nydus-backend-proxy/README.md)

nydusd HTTP服务器，将本地blobs目录用作nydusd的blob后端

```bash
nydus-backend-proxy --blobsdir /path/to/nydus/blobs/dir   #输入blobs的存放目录
```

#### [nydusd](https://github.com/dragonflyoss/image-service/blob/master/docs/nydusd.md)

Linux FUSE user-space daemon，从后端拉取blob数据并解析成原来的文件数据，支持OSS、Localfs、Registry后端，通过配置文件配置存储后端。

```bash
sudo nydusd \
  --config /path/to/nydusd-config.json \       #指定配置文件
  --mountpoint /path/to/mnt \                   #指定挂载点
  --bootstrap /path/to/bootstrap \              #bootstrap文件
  --log-level info
```

### 下载编译

需要安装rust环境，推荐静态编译，避免部署时遇到glibc版本问题。

```bash
git clone https://github.com/dragonflyoss/image-service.git
cd image-service
# build nydusd,nydus-images, x86_64
make static-release
cp target-fusedev/x86_64-unknown-linux-musl/release/nydusd /var/lib/sealer/nydusdfile/clientfile
cp target-fusedev/x86_64-unknown-linux-musl/release/nydus-image /var/lib/sealer/nydusdfile/serverfile
# build nydus-backend-proxy
cd contrib/nydus-backend-proxy
make static-release
cp arget/x86_64-unknown-linux-musl/release/nydus-backend-proxy /var/lib/sealer/nydusdfile/serverfile
```
