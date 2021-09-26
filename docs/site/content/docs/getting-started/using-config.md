+++
title = "Config"
description = "sealer config"
date = 2021-05-01T08:20:00+00:00
updated = 2021-05-01T08:20:00+00:00
draft = false
weight = 22
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Config help you define application config then init the cluster. You can using config to overwrite default config files like helm values"
toc = true
top = false
+++

# Overview

Using config, you can overwrite any config files you want. Like chart values, docker daemon.json, kubeadm config file ...

# Using config overwrite mysql chart values

Append you config metadata into Clusterfile and apply it like this:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-app/my-SAAS-all-inone:latest
  provider: BAREMETAL
...
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: mysql-config
spec:
  path: etc/mysql.yaml
  data: |
       mysql-user: root
       mysql-passwd: xxx
```

`sealer apply -f Clusterfile`

sealer will use the data to overwrite the file `etc/mysql.yaml`

When apply this Clusterfile, sealer will generate some values file for application config. Named etc/mysql-config.yaml etc/redis-config.yaml.

So if you want to use those config, Kubefile is like this:

```yaml
FROM kuberentes:v1.19.9
...
CMD helm install mysql -f etc/mysql-config.yaml
```

# User defined docker systemd config

Of course, you can overwrite other config file in Cloudrootfs you want:

```yaml
.
├── bin
│   ├── conntrack
│   ├── containerd-rootless-setuptool.sh
│   ├── containerd-rootless.sh
│   ├── crictl
│   ├── kubeadm
│   ├── kubectl
│   ├── kubelet
│   ├── nerdctl
│   └── seautil
├── cri
│   ├── containerd
│   ├── containerd-shim
│   ├── containerd-shim-runc-v2
│   ├── ctr
│   ├── docker
│   ├── dockerd
│   ├── docker-init
│   ├── docker-proxy
│   ├── rootlesskit
│   ├── rootlesskit-docker-proxy
│   ├── runc
│   └── vpnkit
├── etc
│   ├── 10-kubeadm.conf
│   ├── Clusterfile  # image default Clusterfile
│   ├── daemon.json
│   ├── docker.service
│   ├── kubeadm-config.yaml
│   └── kubelet.service
├── images
│   └── registry.tar  # registry docker image, will load this image and run a local registry in cluster
├── Kubefile
├── Metadata
├── README.md
├── registry # will mount this dir to local registry
│   └── docker
│       └── registry
├── scripts
│   ├── clean.sh
│   ├── docker.sh
│   ├── init-kube.sh
│   ├── init-registry.sh
│   ├── init.sh
│   └── kubelet-pre-start.sh
└── statics # yaml files, sealer will render values in those files
    └── audit-policy.yml
```

For example, overwrite the docker systemd config:

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: docker-config
spec:
  path: etc/docker.service
  data: |
    [Unit]
    Description=Docker Application Container Engine
    Documentation=https://docs.docker.com
    After=network.target

    [Service]
    Type=notify
    # the default is not to use systemd for cgroups because the delegate issues still
    # exists and systemd currently does not support the cgroup feature set required
    # for containers run by docker
    ExecStart=/usr/bin/dockerd
    ExecReload=/bin/kill -s HUP $MAINPID
    # Having non-zero Limit*s causes performance problems due to accounting overhead
    # in the kernel. We recommend using cgroups to do container-local accounting.
    LimitNOFILE=infinity
    LimitNPROC=infinity
    LimitCORE=infinity
    # Uncomment TasksMax if your systemd version supports it.
    # Only systemd 226 and above support this version.
    #TasksMax=infinity
    TimeoutStartSec=0
    # set delegate yes so that systemd does not reset the cgroups of docker containers
    Delegate=yes
    # kill only the docker process, not all processes in the cgroup
    KillMode=process

    [Install]
    WantedBy=multi-user.target
```