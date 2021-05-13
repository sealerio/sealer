# cloud rootfs

cloud rootfs package all the dependencies when run a kubernetes cluster needs.

```shell script
.
├── bin
│   ├── conntrack
│   ├── crictl
│   ├── kubeadm
│   ├── kubectl
│   ├── kubelet
│   └── seautil
├── cni
│   └── calico
│       └── calico.yaml.tmpl
├── docker
│   ├── containerd
│   ├── containerd-shim
│   ├── containerd-shim-runc-v2
│   ├── ctr
│   ├── docker
│   ├── dockerd
│   ├── docker-init
│   ├── docker-proxy
│   ├── rootlesskit
│   ├── rootlesskit-docker-proxy
│   ├── runc
│   └── vpnkit
├── etc
│   ├── 10-kubeadm.conf
│   ├── Clusterfile # image default Clusterfile
│   ├── docker.service
│   ├── kubeadm.yaml.tmpl
│   └── kubelet.service
├── images
│   └── registry.tar # registry docker image, will load this image and run a local registry in cluster
├── Kubefile
├── Metadata
├── README.md
├── registry # will mount this dir to local registry
│   └── docker
│       └── registry
│           └── v2
├── scripts
│   ├── clean.sh 
│   ├── docker.sh
│   ├── init-registry.sh
│   ├── init.sh
│   └── kubelet-pre-start.sh
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

## Metadata

```shell script
{
  "version": "v1.18.3",
  "arch": "amd64"
}
```

## Hooks

```shell script
FROM kubernetes:1.18.3
COPY preHook.sh /scripts/
```
preHook.sh will execute after init.sh before kubeadm init master0 