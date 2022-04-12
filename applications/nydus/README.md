# Overview

The nydus project implements a user space filesystem on top of a container image format that improves over the current OCI image specification, in terms of container launching speed, image space, and network bandwidth efficiency, as well as data integrity.See [Nydus](https://github.com/dragonflyoss/image-service).

Nydus can help sealer to improve image distribution performance.

## How to use it

Using existing images that support nydus directly, such as:

```shell
sealer run kubernetes-nydus:v1.19.8 --masters xxx.xxx.xxx.xxx  --passwd xxxxxxx
```

or merge nydus image to other images, such as:

```shell
sealer merge kubernetes:v1.19.8 nydus:v1.0.0 -t kubernetes-nydus:1.0.0
```

## How to rebuild it

1. copy nydusdfile dir to /var/lib/sealer

```bash
cp -r nydusdfile /var/lib/sealer
```

2. git clone nydus, buid binary and copy to nydusdfile dir

```bash
cd ~
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

the final nydusdfile dir is as:

```bash
nydusdfile
├── clientfile              #  the files need be scp to inodes to start nydusd
│   ├── clean.sh            # nydusd clean
│   ├── nydusd              # nydusd,Linux FUSE user-space daemon
│   └── start.sh            # start nydusd
└── serverfile              # the files for nydusdserver
    ├── nydus-backend-proxy # A simple HTTP server to serve local directory as a blob backend for nydusd
    ├── nydusblobs          # nydus blobs dir
    ├── nydus-image         # Convert dir into a nydus format container image generating meta part file and data part file respectively
    ├── Rocket.toml         # Roket config file of nydusd http server
    ├── serverclean.sh      # nydusd http server clean
    └── serverstart.sh      # convert nydus images and start nydusd http server
```

3. modify Metadata

set "NydusFlag":true, such as

```bash
# vim /var/lib/sealer/Metadata
{
  "version": "v1.19.8",
  "arch": "x86_64",
  "NydusFlag":true
}
```

4. build new image

```bash
cd /var/lib/sealer
sealer build -t {Your Image Name} -f Kubefile .
```
