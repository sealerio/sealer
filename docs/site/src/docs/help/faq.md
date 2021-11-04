# FAQ

This section is mean to answer the most frequently asked questions about sealer. And it will be updated regularly.

## Sealer failed to cache docker image.

### Build environment check.

The first thing need to know is that we hacked docker to support cache docker images,so only "sealer docker" will have
the cache feature.

1. run `cat /etc/hosts` to check the hosts file, examples like "172.18.0.2 sea.hub",make sure "sea.hub" exist in your
   host.

```yaml
172.18.0.2 sealer-master-4a04d36e4d
172.18.0.2 apiserver.cluster.local
172.18.0.2 sea.hub
```

2. run `docker version` to check the server version, examples like "Version: 19.03.14-sealer"

```yaml
[ root@sealerhost1 ~]# docker version
    Client: Docker Engine - Community
      Version: 19.03.14
      API version: 1.40
      Go version: go1.13.15
      Git commit: 5eb3275
      Built: Tue Dec  1 19:14:24 2020
      OS/Arch: linux/amd64
      Experimental: false

      Server:
        Engine:
          Version: 19.03.14-sealer
          API version: 1.40 (minimum version 1.12)
          Go version: go1.13.15
          Git commit: 711cc111cf
          Built: Wed Jun  2 09:07:15 2021
          OS/Arch: linux/amd64
          Experimental: false
        containerd:
          Version: 1.4.11
          GitCommit: 5b46e404f6b9f661a205e28d59c982d3634148f8
        runc:
          Version: 1.0.2
          GitCommit: v1.0.2-0-g52b36a2
        docker-init:
          Version: 0.18.0
          GitCommit: fec3683

```

3. run `cat /etc/docker/daemon.json` to see the server config,examples like below

```json
{
  "debug": true,
  "max-concurrent-downloads": 20,
  "log-driver": "json-file",
  "log-level": "info",
  "log-opts": {
    "max-size": "10m",
    "max-file": "3"
  },
  "insecure-registries": [
    "sea.hub",
    "sea.hub:5000"
  ],
  "mirror-registries": [
    {
      "domain": "*",
      "mirrors": [
        "http://sea.hub:5000"
      ]
    }
  ],
  "data-root": "/var/lib/docker"
}
```

make sure the local docker config meet the required.

4. check the build log,whether if the docker pull process happened,examples like below output. if no docker pull process
   happened, check your build context,make sure it is right.

```text
2021-11-01 16:31:56 [INFO] [build_image.go:68] run build layer: COPY imageList manifests
v1.6.0: Pulling from kcr-3rd/csi-provisioner
495922835b01: Pull complete
Digest: sha256:78e3393f5fd5ff6c1e5dada2478cfa456fb7164929e573cf9a87bf6532730679  9.247MB/18.75MB
Status: Downloaded newer image for registry.cn-shanghai.aliyuncs.com/kcr-3rd/csi-provisioner:v1.6.0
2021-11-01 16:32:31 [INFO] [images.go:83] success to pull docker image: registry.cn-shanghai.aliyuncs.com/kcr-3rd/csi-provisioner:v1.6.0
v1.1.0: Pulling from kcr-3rd/csi-node-driver-registrar
4e1edcbff92b: Pull complete
bc586e01076f: Pull complete
Digest: sha256:de74d65da5b3ab7875e1c0259633a155762f160d3f2f64ed0a3d3197b1da0c00
Status: Downloaded newer image for registry.cn-shanghai.aliyuncs.com/kcr-3rd/csi-node-driver-registrar:v1.1.0
2021-11-01 16:32:44 [INFO] [images.go:83] success to pull docker image: registry.cn-shanghai.aliyuncs.com/kcr-3rd/csi-node-driver-registrar:v1.1.0
```

if no docker registry started process,please contact the sealer team.

```text
+ cd .
+ REGISTRY_PORT=5000
+ VOLUME=/var/lib/sealer/tmp/.DTmp-879039835/registry
+ container=sealer-registry
+ mkdir -p /var/lib/sealer/tmp/.DTmp-879039835/registry
+ docker load -q -i ../images/registry.tar
Loaded image: registry:2.7.1
+ docker rm sealer-registry -f
Error: No such container: sealer-registry
+ true
+ docker run -d --restart=always --net=host --name sealer-registry -v /var/lib/sealer/tmp/.DTmp-879039835/registry:/var/lib/registry registry:2.7.1
fd12489bb01805f3f700cedc4b41b236b88e529fea747ee0e33cc6a556158919
```

### Build result check.

we suggest you have a clean machine to run lite build,if the image already exist in the hosts,it will not cache to the
registry. you can run `docker system prune -a` to clean the host images.

1. run `sealer inspect {your iamge name}`, check the last layer where its type is "type: BASE",and value is "registry
   cache". some part of the image spec as blow:

```yaml
spec:
  id: bfb2810f9ad176cb9bc39e4a98d6319ea8599fa22a0911a98ae0b3e86e96b0a4
  layers:
    - id: sha256:c1aa4aff818df1dd51bd4d28decda5fe695bea8a9ae6f63a8dd5541c7640b3d6
      type: COPY
      value: . .
    - id: sha256:931c62b6d883c3d8f53eae21220ac7370ae0a72f70157e4be90022b70aab77b0
      type: COPY
      value: Clusterfile etc/
    - type: BASE
      value: registry cache
    - id: sha256:991491d3025bd1086754230eee1a04b328b3d737424f1e12f708d651e6d66860
      type: COPY
      value: etc .
    - type: CMD
      value: kubectl apply -f etc/tigera-operator.yaml
    - type: CMD
      value: kubectl apply -f etc/custom-resources.yaml
    - id: sha256:d2c72ca0b588720f7ad32b2ecde280f5912acd57c61d285b839a4193da78d90d
      type: BASE
      value: registry cache
  platform: { }
  sealer_version: latest
status: { }
```

2. if not existed,there is no docker image cached in the cloud image,check your Kubefile and docker config. If you use
   sealer docker,please create an issue for sealer team.
3. if existed, check all image layers to
   run ` ls -l /var/lib/sealer/data/overlay2/{layer id}/registry/docker/registry/v2/repositories/{your image}/_layers/sha256/`

```shell
[root@sealerhost1 ~]# ls -l /var/lib/sealer/data/overlay2/d2c72ca0b588720f7ad32b2ecde280f5912acd57c61d285b839a4193da78d90d/registry/docker/registry/v2/repositories/calico/node/_layers/sha256/
drwxr-xr-x. 2 root root 18 Oct  9 16:47 954e0bcac799e3a9c4367e84c5374db6690dc567d78515d142158d74db8dd3bd
drwxr-xr-x. 2 root root 18 Oct  9 16:47 c4d75af7e098eef5142c65b8fab5b3ff0e5c614ce72e91b8c44ba20249b596ef
drwxr-xr-x. 2 root root 18 Oct  9 16:50 d226bad0de34eda32554694086f79b503404476ddb8c68089cbd62c33515b637
```

check all image layers content to
run `ls -l /var/lib/sealer/data/overlay2/{layer id}/registry/docker/registry/v2/blobs/sha256/`

```shell
[root@sealerhost1 ~]# ls -l /var/lib/sealer/data/overlay2/d2c72ca0b588720f7ad32b2ecde280f5912acd57c61d285b839a4193da78d90d/registry/docker/registry/v2/blobs/sha256
total 0
drwxr-xr-x. 3 root root  78 Oct  9 16:44 0d
drwxr-xr-x. 3 root root  78 Oct  9 16:45 95
drwxr-xr-x. 3 root root  78 Oct  9 16:51 59
drwxr-xr-x. 3 root root  78 Oct  9 16:50 c4
drwxr-xr-x. 3 root root  78 Oct  9 16:45 d2
```

make sure all the layers and layer content exist at the same time in this cloud images. if only part of them, please
clean all docker images of your build machine, and run sealer build steps again.

4. use docker inspect to check all the cached image layer.

get all docker image layer :

examples:

`docker inspect --format='{{json .RootFS}}' 8c72b944d569`

output:

```json
{
  "Type": "layers",
  "Layers": [
    "sha256:4b0a2b20e92dcbf057d10806b8aa690b26d1d3dd33b0fc63d838f4acaf23bd07",
    "sha256:3e1226931b2290a838eec9bbbd911e4f5da535a447f3f96481017b41bf9c0259"
  ]
}
```

choose "4b0a2b20e92dcbf057d10806b8aa690b26d1d3dd33b0fc63d838f4acaf23bd07" as examples to read the registry layer id from
from distribution directory.

`cat /var/lib/docker/image/overlay2/distribution/v2metadata-by-diffid/sha256/4b0a2b20e92dcbf057d10806b8aa690b26d1d3dd33b0fc63d838f4acaf23bd07`

```json
[
  {
    "Digest": "sha256:5d3835484afecc78dccfa2f7d4fcf273aacfe0c7600b957314e38488f3942045",
    "SourceRepository": "docker.io/library/traefik",
    "HMAC": ""
  }
]
```

check with the sealer image cache,we can see "5d3835484afecc78dccfa2f7d4fcf273aacfe0c7600b957314e38488f3942045" show
below "_layers" directory.

"8c72b944d56909f092c54c2b0804002f5501a61b7f4444e03574c0ff3455d657" is the imagedb ,not the image content layer.

```text
library
        └── traefik
            ├── _layers
            │   └── sha256
            │       ├── 0feefa6e9e49547c30d8edd85bbe6116ad1107ec138af7b22af9e087d759de0c
            │       │   └── link
            │       ├── 5d3835484afecc78dccfa2f7d4fcf273aacfe0c7600b957314e38488f3942045
            │       │   └── link
            │       └── 8c72b944d56909f092c54c2b0804002f5501a61b7f4444e03574c0ff3455d657
            │           └── link
            ├── _manifests
            │   ├── revisions
            │   │   └── sha256
            │   │       ├── d1264267935f35aa1070a840d24bfc6bb7f55efb49949589b049f82a4c5967f4
            │   │       │   └── link
            │   │       └── d277007b55a8a8d972b1983ef11387d05f719821a2d2e23e8fa06ac5081a302f
            │   │           └── link
```