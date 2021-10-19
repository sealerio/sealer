# YRCloudFile

[YRCloudFile](https://www.yanrongyun.com) is a high performance distributed filesystem, which can be deployed either on-premise or off-premise, and fully support the Kubernetes.

By integrated with sealer, you can deploy the YRCloudFile and kubernetes seamlessly, just by one Clusterfile.

## Concept
oss: data storage node role, and contains oss disks; all the oss_disks should be even number.

mds: meta storage node role ,and contains mds disks; all the mds_disks should be even number.

mgr: storage management node role;nodes with mgr role should be 3 or 1.

## Using YRCloudFile CloudImage

This image contains all the dependencies. and for Supported Kernel, please check the [compatible-kernels.txt]

Clusterfile:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  certSANS:
  - aliyun-inc.com
  - 10.0.0.2
  image: registry.cn-qingdao.aliyuncs.com/yrcloudfile/yrcloudfile:latest
  masters:
...

```

for your target hosts, either virtual machine or bare-metal, generated the following content.

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: yrcloudfile.yml
spec:
  path: etc/yrcloudfile.yml
  data: |
    hosts:
    - hostname: node28.yr
      mds_disks: [/dev/sdb,/dev/sdc]
      mgmt_network: {device: ens224, ipaddr: 192.168.13.28}
      oss_disks: [/dev/sdd, /dev/sde]
      password: Passw0rd
      roles: [mds, oss, mgr]
    - hostname: node29.yr
      mds_disks: [/dev/sdb,/dev/sdc]
      mgmt_network: {device: ens224, ipaddr: 192.168.13.29}
      oss_disks: [/dev/sdd, /dev/sde]
      password: Passw0rd
      roles: [mds, oss, mgr]
    - hostname: node35.yr
      mds_disks: [/dev/sdb,/dev/sdc]
      mgmt_network: {device: ens224, ipaddr: 192.168.13.35}
      oss_disks: [/dev/sdd, /dev/sde]
      password: Passw0rd
      roles: [mds, oss, mgr]
    mgmt_cidr: 192.168.0.0/16  # cidr for the hosts ip address
    mode: install
    owner: sealer
    http_port: 8080
    etcd_port: 2479
    etcd_peer_port: 2480
    mount_path: /mnt/pfs
    storage_cidr: 192.168.0.0/16  # keep it as same as mgmt_cidr for now
    version: 6.6.2
```

Here the 'my-cluster' must be the SAME with cluster name

The last two parameters of shell command should be ip address of mgmt_network for one host which with mgr role, and the http_port defined the Config file

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PreInstall
  data: |
    cd /var/lib/sealer/data/my-cluster/rootfs/yrcloudfile && chmod 0755 all.sh && sh -x all.sh 192.168.13.28 8080
```

Full example ,please refer to example/Clusterfile

```
sealer apply -f Clusterfile
```
