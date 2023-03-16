# cloud rootfs

cloud rootfs will package all the dependencies refers to the kubernetes cluster requirements

```shell script
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
├── Metadata
├── imageList
├── scripts
│   ├── docker.sh
│   ├── uninstall-docker.sh
│   ├── containerd.sh
│   ├── uninstall-containerd.sh
│   ├── init-kube.sh
│   └── init-registry.sh
└── statics # yaml files, sealer will render values in those files
    └── audit-policy.yml
```

Using cloud rootfs to build a base cloudImage:

```shell script
FROM scratch
COPY . .
```

```shell script
sealer build -t kuberntes:v1.18.3 .
```

If you put any file in the $rootfs and then build a cluster image, then when you use the image to deploy, sealer will ensure that these files will be put on the same relative path in all nodes' $rootfs.

## Put your binaries in $rootfs/bin
This directory is for binaries.

## [Deprecated] Put your CRI packages in $rootfs/cri
This directory is for CRI.

## Put your configurations in $rootfs/etc/
This directory is for configurations.

### Use env render
All file with suffix '.tmpl' will be rendered using ClusterFile's env. For example, a file named XABC.yaml.tmpl will be rendered and saved into file XABC.yaml.

[Env render](https://github.com/sealerio/sealer/blob/main/docs/design/global-config.md#global-configuration)

## [Deprecated] Put your registry image tar in $rootfs/images
This directory is for registry image.

## Write your kubernetes metadata in Metadata

For example:

```shell script
{
  "version": "v1.18.3",
  "arch": "amd64"
}
```

## imageList
Write the image list which your want involved in ClusterImage into imageList, for example:

```
docker.io/alpine:3.17.1
registry.k8s.io/pause:3.5
registry.k8s.io/kube-apiserver:v1.22.17
```

## scripts
This directory is for scripts.

The follow files are reserved by the system, you can override them to realize the functions you need:

- docker.sh, for installing Docker.
- uninstall-docker.sh, for cleaning Docker.
- containerd.sh, for installing Containerd.
- uninstall-containerd.sh, for cleaning Docker.
- init-registry.sh, for installing registry.
- init-kube.sh, for installing kube*.

If you want to add a new CRI type, you can put packages in this directory, and add a couple of scripts named ${CRI_NAME}.sh and uninstall-${CRI_NAME}.sh in directory $rootfs/scripts:

- ${CRI_NAME}.sh: used to install this CRI, and write the CRI socket info into /etc/sealerio/cri/socket-path
- uninstall-${CRI_NAME}.sh: used to uninstall this CRI

Users can specify container runtime type to use customized CRI:

Clusterfile:

```yaml
apiVersion: sealer.io/v1
kind: Cluster
spec:
  containerRuntime:
    type: ${CRI_NAME}
...
```

## Hooks

```shell script
FROM kubernetes:1.18.3
COPY preHook.sh /scripts/
```

preHook.sh will execute after init.sh before kubeadm init master0
