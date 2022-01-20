# FAQ

This section is mean to answer the most frequently asked questions about sealer. And it will be updated regularly.

## How to clean host environment manually when sealer apply failed.

in some case ,when you failed to run sealer apply ,and the hints show a little that is not enough to use, this section
will guild you how to clean your host manually.

you may follow the below clean steps when run kubeadm init failed.

### umount rootfs or apply mount if it existed

```shell
df -h | grep sealer
overlay          40G  7.3G   31G  20% /var/lib/sealer/data/my-cluster/rootfs
```

umount examples:

```shell
umount /var/lib/sealer/data/my-cluster/rootfs
```

## delete rootfs directory if it existed

```shell
rm -rf /var/lib/sealer/data/my-cluster
```

## delete kubernetes directory if it existed

```shell
rm -rf /etc/kubernetes
rm -rf /etc/cni
rm -rf /opt/cni
```

## delete docker registry if it existed

```shell
docker ps
docker rm -f -v sealer-registry
```

you may follow the below clean steps if your cluster is up.

## kubeadm reset

```shell
kubeadm reset -f
```

## delete kube config and kubelet if it existed

```shell
rm -rf $HOME/.kube/config
rm -rf ~/.kube/ && rm -rf /etc/kubernetes/ && \
rm -rf /etc/systemd/system/kubelet.service.d && rm -rf /etc/systemd/system/kubelet.service && \
rm -rf /usr/bin/kube* && rm -rf /usr/bin/crictl && \
rm -rf /etc/cni && rm -rf /opt/cni && \
rm -rf /var/lib/etcd && rm -rf /var/etcd
```