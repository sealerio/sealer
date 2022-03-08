# 自定义Cloud image

## 自定义CloudRootfs

运行kubernetes集群所需的所有文件。

其中包含：

* Bin 文件，如 docker、containerd、crictl、kubeadm、kubectl...
* 配置文件，如 kubelet systemd config、docker systemd config、docker daemon.json...
* 注册docker镜像。
* 一些元数据，例如 Kubernetes 版本
* 注册表文件，包含所有的docker镜像，比如kubernetes核心组件docker镜像...* Scripts, some shell script using to install docker and kubelet... sealer will call init.sh and clean.sh.
* 其他静态文件

rootfs 树状图

```
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
│   └── docker.tar.gz # cri 二进制文件包括 docker、containerd、runc。
├── etc
│   ├── 10-kubeadm.conf
│   ├── Clusterfile  # 镜像默认集群文件
│   ├── daemon.json # docker 守护进程配置文件。
│   ├── docker.service
│   ├── kubeadm.yml # kubeadm config 包括 Cluster Configuration、JoinConfiguration 等。
│   ├── kubelet.service
│   ├── registry_config.yml # docker 注册表配置，包括存储根目录和 http 相关配置。
│   └── registry.yml # 如果用户想自定义用户名和密码，可以覆盖这个文件。
├── images
│   └── registry.tar  # 注册docker镜像，将加载此镜像并在集群中运行本地注册表
├── Kubefile
├── Metadata
├── README.md
├── registry # 将此目录挂载到本地注册表
│   └── docker
│       └── registry
├── scripts
│   ├── clean.sh
│   ├── docker.sh
│   ├── init-kube.sh
│   ├── init-registry.sh
│   ├── init.sh
│   └── kubelet-pre-start.sh
└── statics # yaml文件, sealer 将渲染这些文件中的值
    └── audit-policy.yml
```

### 如何获取 CloudRootfs

1. 拉取基础镜像 `sealer pull kubernetes:v1.19.8-alpine`
2. 查看镜像层信息 `sealer inspect kubernetes:v1.19.8-alpine`
3. 进入BaseImage层 `ls /var/lib/sealer/data/overlay2/{layer-id}`

您将找到 CloudRootfs 层。

### 构建自己的 CloudRootfs

您可以在 CloudRootfs 中编辑您想要的任何文件，例如您想定义自己的 docker daemon.json，只需编辑它并构建一个新的 CloudImage。

```shell script
FROM scratch
COPY . .
```

```shell script
sealer build -t user-defined-kubernetes:v1.19.8 .
```

然后，您可以将此镜像用作 BaseImage。

### 覆盖 CloudRootfs 文件

有时您不想关心 CloudRootfs 上下文，但需要自定义一些配置。

您可以使用 `kubernetes:v1.19.8` 作为 BaseImage，并使用自己的配置文件覆盖 CloudRootfs 中的默认文件。

例如：daemon.json 是您的 docker 引擎配置，使用它来覆盖默认配置：

```shell script
FROM kubernetes:v1.19.8
COPY daemon.json etc/
```

```shell script
sealer build -t user-defined-kubernetes:v1.19.8 .
```

## 构建cloud image

### 使用特定目录构建

#### image目录

保存容器镜像的目录，该目录下的离线镜像会在sealer运行时加载到内置注册表中。

示例：将离线 tar 文件复制到此目录。

`COPY mysql.tar images`

#### plugin目录

插件文件保存目录，该目录下的插件文件会在sealer运行时加载到运行界面。

示例：将插件配置文件复制到此目录。

插件配置：shell.yaml：

```
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: taint
spec:
  type: SHELL
  action: PostInstall
  on: node-role.kubernetes.io/master=
  data: |
     kubectl taint nodes --all node-role.kubernetes.io/master-
```

`COPY shell.yaml plugins`

#### charts目录

保存charts包的目录，sealer构建时会解析该目录下的charts文件，下载并保存对应的容器镜像。

示例：将 nginx charts复制到此目录。

`COPY nginx charts`

#### manifests目录

保存yaml文件或“imageList”文件的目录，sealer构建时会解析该目录下的yaml和“imageList”文件，下载并保存对应的容器镜像。

示例：将“imageList”文件复制到此目录。

```shell
[root@iZbp143f9driomgoqx2krlZ build]# cat imageList
busybox
```

`COPY imageList manifests`

示例：将仪表板 yaml 文件复制到此目录。

`COPY recommend.yaml manifests`

### 自定义私有registry

Sealer对docker registry进行了优化和扩展，使其可以同时支持多个域名的代理缓存和多个私有registry。

在构建过程中，会出现使用需要身份验证的私有registry的情况。在这种情况下，镜像缓存需要 docker 的身份验证。在执行构建操作之前，可以先通过以下命令进行登录操作：

```shell
sealer login registry.com -u username -p password
```

另一种依赖场景，kubernetes节点通过sealer内置registry代理私有registry，私有registry需要认证，可以通过自定义registryconfig配置。参考 [registry config](../../../../design/docker-image-cache.md)

您可以通过定义 Kubefile 来自定义注册表配置：

```shell
FROM kubernetes:v1.19.8
COPY registry_config.yaml etc/
```

### 自定义 kubeadm 配置

Sealer 会将默认配置替换为 $Rootfs/etc/kubeadm.yml 中的自定义配置文件。

#### 示例：使用 Docker Unix socket的自定义配置。

1. 自定义 kubeadm 初始化配置：

```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
localAPIEndpoint:
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

2. 自定义 kubeadm join 配置：

```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
caCertPath: /etc/kubernetes/pki/ca.crt
discovery:
  timeout: 5m0s
nodeRegistration:
  criSocket: /var/run/dockershim.sock
controlPlane:
  localAPIEndpoint:
    bindPort: 6443
```

3. 构建您自己的云映像，使用自定义配置覆盖默认配置。请注意，文件名“kubeadm.yml”是固定的：

```yaml
#Kubefile
FROM kubernetes-clusterv2:v1.19.8
COPY kubeadm.yml etc
```

> sealer build -t user-define-kubeadm-kubernetes:v1.19.8 .

#### 包含完整内容的默认 kubeadm 配置文件：

选择 kubeadm.yml 的任何部分进行自定义：

```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
localAPIEndpoint:
  # advertiseAddress: 192.168.2.110
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock

---
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
#controlPlaneEndpoint: "apiserver.cluster.local:6443"
imageRepository: sea.hub:5000/library
networking:
  # dnsDomain: cluster.local
  podSubnet: 100.64.0.0/10
  serviceSubnet: 10.96.0.0/22
apiServer:
  #  certSANs:
  #    - 127.0.0.1
  #    - apiserver.cluster.local
  #    - aliyun-inc.com
  #    - 10.0.0.2
  #    - 10.103.97.2
  extraArgs:
    #    etcd-servers: https://192.168.2.110:2379
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
    audit-policy-file: "/etc/kubernetes/audit-policy.yml"
    audit-log-path: "/var/log/kubernetes/audit.log"
    audit-log-format: json
    audit-log-maxbackup: '10'
    audit-log-maxsize: '100'
    audit-log-maxage: '7'
    enable-aggregator-routing: 'true'
  extraVolumes:
    - name: "audit"
      hostPath: "/etc/kubernetes"
      mountPath: "/etc/kubernetes"
      pathType: DirectoryOrCreate
    - name: "audit-log"
      hostPath: "/var/log/kubernetes"
      mountPath: "/var/log/kubernetes"
      pathType: DirectoryOrCreate
    - name: localtime
      hostPath: /etc/localtime
      mountPath: /etc/localtime
      readOnly: true
      pathType: File
controllerManager:
  extraArgs:
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
    experimental-cluster-signing-duration: 876000h
  extraVolumes:
    - hostPath: /etc/localtime
      mountPath: /etc/localtime
      name: localtime
      readOnly: true
      pathType: File
scheduler:
  extraArgs:
    feature-gates: TTLAfterFinished=true,EphemeralContainers=true
  extraVolumes:
    - hostPath: /etc/localtime
      mountPath: /etc/localtime
      name: localtime
      readOnly: true
      pathType: File
etcd:
  local:
    extraArgs:
      listen-metrics-urls: http://0.0.0.0:2381
---
apiVersion: kubeproxy.config.k8s.io/v1alpha1
kind: KubeProxyConfiguration
mode: "ipvs"
ipvs:
  excludeCIDRs:
    - "10.103.97.2/32"

---
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration
authentication:
  anonymous:
    enabled: false
  webhook:
    cacheTTL: 2m0s
    enabled: true
  x509:
    clientCAFile: /etc/kubernetes/pki/ca.crt
authorization:
  mode: Webhook
  webhook:
    cacheAuthorizedTTL: 5m0s
    cacheUnauthorizedTTL: 30s
cgroupDriver:
cgroupsPerQOS: true
clusterDomain: cluster.local
configMapAndSecretChangeDetectionStrategy: Watch
containerLogMaxFiles: 5
containerLogMaxSize: 10Mi
contentType: application/vnd.kubernetes.protobuf
cpuCFSQuota: true
cpuCFSQuotaPeriod: 100ms
cpuManagerPolicy: none
cpuManagerReconcilePeriod: 10s
enableControllerAttachDetach: true
enableDebuggingHandlers: true
enforceNodeAllocatable:
  - pods
eventBurst: 10
eventRecordQPS: 5
evictionHard:
  imagefs.available: 15%
  memory.available: 100Mi
  nodefs.available: 10%
  nodefs.inodesFree: 5%
evictionPressureTransitionPeriod: 5m0s
failSwapOn: true
fileCheckFrequency: 20s
hairpinMode: promiscuous-bridge
healthzBindAddress: 127.0.0.1
healthzPort: 10248
httpCheckFrequency: 20s
imageGCHighThresholdPercent: 85
imageGCLowThresholdPercent: 80
imageMinimumGCAge: 2m0s
iptablesDropBit: 15
iptablesMasqueradeBit: 14
kubeAPIBurst: 10
kubeAPIQPS: 5
makeIPTablesUtilChains: true
maxOpenFiles: 1000000
maxPods: 110
nodeLeaseDurationSeconds: 40
nodeStatusReportFrequency: 10s
nodeStatusUpdateFrequency: 10s
oomScoreAdj: -999
podPidsLimit: -1
port: 10250
registryBurst: 10
registryPullQPS: 5
rotateCertificates: true
runtimeRequestTimeout: 2m0s
serializeImagePulls: true
staticPodPath: /etc/kubernetes/manifests
streamingConnectionIdleTimeout: 4h0m0s
syncFrequency: 1m0s
volumeStatsAggPeriod: 1m0s
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: JoinConfiguration
caCertPath: /etc/kubernetes/pki/ca.crt
discovery:
  timeout: 5m0s
nodeRegistration:
  criSocket: /var/run/dockershim.sock
controlPlane:
  localAPIEndpoint:
    bindPort: 6443
```