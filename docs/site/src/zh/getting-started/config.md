# 使用Config功能

使用 config，你可以覆盖或合并任何你想要的配置文件。像chart values、docker daemon.json、kubeadm 配置文件等。

## 覆盖配置

### 使用Config覆盖重写*calico*自定义配置

以镜像`registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`为例:

```yaml
# 默认calico配置文件custom-resources.yaml：
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    ipPools:
    - blockSize: 26
      cidr: 100.64.0.0/10
      encapsulation: IPIP
      natOutgoing: Enabled
      nodeSelector: all()
    nodeAddressAutodetectionV4:
      interface: "eth.*|en.*"
```

如果不满足默认IP自动检测规则或需要修改CIDR ，则将修改后的配置元数据附加到 Clusterfile 并应用：

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  ssh:
    passwd: xxx
  hosts:
    - ips: [192.168.0.2,192.168.0.3,192.168.0.4]
      roles: [master]
    - ips: [192.168.0.5]
      roles: [node]
...
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: calico
spec:
  path: etc/custom-resources.yaml
  data: |
    apiVersion: operator.tigera.io/v1
    kind: Installation
    metadata:
      name: default
    spec:
      calicoNetwork:
        ipPools:
        - blockSize: 26
          cidr: 100.64.0.0/10 #需与kubeadm配置中cidr一致
          encapsulation: IPIP
          natOutgoing: Enabled
          nodeSelector: all()
        nodeAddressAutodetectionV4:
          interface: "eth*|en*" #将IP自动检测规则改成相应符合的规则
```

`sealer apply -f Clusterfile`

### 使用config覆盖 mysql chart values

添加mysql配置元数据到Clusterfile并应用:

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

sealer 将使用该数据覆盖文件 `etc/mysql.yaml`

应用此 Clusterfile 时，sealer 将为应用程序配置生成一些值文件。命名该配置为 etc/mysql-config.yaml etc/redis-config.yaml。

所以如果你想要使用该配置，Kubefile例如：

```yaml
FROM kuberentes:v1.19.9
...
CMD helm install mysql -f etc/mysql-config.yaml
```

### 用户定义的 docker systemd 配置

当然，你可以覆盖你想要的rootfs中的其他配置文件:

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
│   ├── Clusterfile  # 镜像默认 Clusterfile
│   ├── daemon.json
│   ├── docker.service
│   ├── kubeadm-config.yaml
│   └── kubelet.service
├── images
│   └── registry.tar  # registry docker 镜像，将加载此镜像并在集群中运行本地registry
├── Kubefile
├── Metadata
├── README.md
├── registry # registry data数据，此目录将挂载到本地registry
│   └── docker
│       └── registry
├── scripts
│   ├── clean.sh
│   ├── docker.sh
│   ├── init-kube.sh
│   ├── init-registry.sh
│   ├── init.sh
│   └── kubelet-pre-start.sh
└── statics
    └── audit-policy.yml
```

例如，覆盖 docker systemd 配置:

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

## 合并配置（yaml格式）

### 使用Config功能合并*calico*自定义配置

以镜像`registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`为例:

合并配置只需要关心需要修改的部分，以合并的方式修改calicoIP自动检测规则配置：

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
  ssh:
    passwd: xxx
  hosts:
    - ips: [192.168.0.2,192.168.0.3,192.168.0.4]
      roles: [master]
    - ips: [192.168.0.5]
      roles: [node]
...
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: calico
spec:
  strategy: merge #默认为覆盖形式，merge表示合并config
  path: etc/custom-resources.yaml
  data: |
    spec:
      calicoNetwork:
        nodeAddressAutodetectionV4:
          interface: "enp*" #将IP自动检测规则改成相应符合的规则
```

`sealer apply -f Clusterfile`

sealer启动后会合并原配置文件$/rootfs/etc/custom-resources.yaml并修改:

```yaml
apiVersion: operator.tigera.io/v1
kind: Installation
metadata:
  name: default
spec:
  calicoNetwork:
    ipPools:
    - blockSize: 26
      cidr: 100.64.0.0/10
      encapsulation: IPIP
      natOutgoing: Enabled
      nodeSelector: all()
    nodeAddressAutodetectionV4:
      interface: "enp*"
```

>spec.calicoNetwork.nodeAddressAutodetectionV4.interface="enp*"修改成功。