# cloud rootfs

cloud rootfs will package all the dependencies refers to the kubernetes cluster requirements

```shell script
.
├── README.md # Metadata will be see in the `sealer help`
├── bin # binaries will be installed at all nodes' /usr/local/bin
│   ├── conntrack
│   ├── containerd-rootless-setuptool.sh
│   ├── containerd-rootless.sh
│   ├── crictl
│   ├── kubeadm
│   ├── kubectl
│   ├── kubelet
│   ├── nerdctl
│   ├── kubelet-pre-start.sh
│   ├── helm
│   └── seautil
├── etc # configs will be put into all nodes' sealer-rootfs
│   ├── 10-kubeadm.conf
│   ├── daemon.json
│   ├── docker.service
│   ├── audit-policy.yml
│   ├── kubeadm-config.yaml
│   ├── kubeadm-config.yaml.tmpl # a.b.c.tmpl will be rendered using envs and rename to a.b.c
│   └── kubelet.service
├── plugins # plugins can run on some hooks, such as pre-init-host, post-install, see more in the plugins documentation
│   └── disk_init_shell_plugin.yaml
├── scripts # scripts can use all ClusterFile's env as Linux env variables
│   ├── init-kube.sh # initialize kube* binaries on target hosts
│   ├── clean-kube.sh # remove kube* binaries from target hosts
│   ├── init-container-runtime.sh # initialize container runtime binaries on target hosts
│   ├── clean-container-runtime.sh # remove container runtime binaries on target hosts
│   └── init-registry.sh # initialize registry on local registry's deploy-hosts
└── applications # applications put here
│   ├── redis
│   └── mysql
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

## Registry

registry container name must be 'sealer-registry'
