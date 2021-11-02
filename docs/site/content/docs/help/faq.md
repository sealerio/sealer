+++
title = "FAQ"
description = "Answers to frequently asked questions."
date = 2021-05-01T19:30:00+00:00
updated = 2021-05-01T19:30:00+00:00
draft = false
weight = 30
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Answers to frequently asked questions."
toc = true
top = false
+++

# Introduction

This section is mean to answer the most frequently asked questions about sealer. And it will be updated regularly.

## Sealer failed to cache docker image.

### Build environment check.

The first thing need to know is that we hacked docker to support cache docker images,so only "sealer docker" will have
the cache feature.

1. run `docker version` to check the server version, examples like "Version: 19.03.14-sealer"

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

2. run `cat /etc/docker/daemon.json` to see the server config,examples like below

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

3.check the build log,whether if the docker pull process happened,examples like below output. if no docker pull process
happened, check your build context,make sure it is right.

![img_1.png](img_1.png)

if no docker registry started process,please contact the sealer team.

![img_2.png](img_2.png)

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