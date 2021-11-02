# 镜像构建

> 就像使用Dockerfile来构建容器镜像一样， 我们可以通过Kubefile来定义一个sealer的集群镜像。我们可以使用和Dockerfile一样的指令来定义一个可离线部署的交付镜像。

## Kubefile样例

For example:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this CloudImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

>如果使用ARM64机器，FROM指令使用registry.cn-beijing.aliyuncs.com/sealer-io/kubernetes-arm64:v1.19.7作为基础镜像。

## Kubefile指令说明

### FROM指令

FROM: 引用一个基础镜像，并且Kubefile中第一条指令必须是FROM指令。若基础镜像为私有仓库镜像，则需要仓库认证信息，另外 sealer社区提供了官方的基础镜像可供使用。

> 命令格式：FROM {your base image name}

使用样例：

例如上面示例中,使用sealer 社区提供的`kubernetes:v1.19.8`作为基础镜像。

`FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`

### COPY指令

COPY: 复制构建上下文中的文件或者目录到rootfs中。

集群镜像文件结构均基于[rootfs结构](../../docs/api/cloudrootfs.md),默认的目标路径即为rootfs，且当指定的目标目录不存在时会自动创建。

如需要复制系统命令，例如复制二进制文件到操作系统的$PATH，则需要复制到rootfs中的bin目录，该二进制文件会在镜像构建和启动时，自动加载到系统$PATH中。

> 命令格式：COPY {src dest}

使用样例：

复制mysql.yaml到rootfs目录中。

`COPY mysql.yaml .`

复制可执行文件到系统$PATH中。

`COPY helm ./bin`

### RUN指令

RUN: 使用系统shell执行构建命令，仅在build时运行，可接受多个命令参数，且构建时会将命令执行产物结果保存在镜像中。若系统命令不存在则会构建失败,则需要提前执行COPY指令，将命令复制到镜像中。

> 命令格式：RUN {command args ...}

使用样例：

例如上面示例中,使用wget 命令下载一个kubernetes的dashboard。

`RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml`

### CMD指令

CMD: 与RUN指令格式类似，使用系统shell执行构建命令。但CMD指令会在镜像启动的时候执行，一般用于启动和配置集群使用。另外与Dockerfile中CMD指令不同，一个kubefile中可以有多个CMD指令。

> 命令格式：CMD {command args ...}

使用样例：

例如上面示例中,使用 kubectl 命令安装一个kubernetes的dashboard。

`CMD kubectl apply -f recommended.yaml`

## 执行构建命令解析：

```bigquery
$ sealer build -f Kubefile -t my-kubernetes:v1.19.8 -b cloud .
 -f : 指定Kubefile路径，默认为当前路径下Kubefile
 -t : 指定构建产出镜像的名称
 -b : 指定构建模式[cloud |container |lite] #默认为cloud
 .  : build上下文，指定为当前路径
```

## build类型

> 针对不同的业务需求场景，sealer build 目前支持3种构建方式。

### 1.cloud build

> 默认的build类型。基于云服务（目前仅支持阿里云， 欢迎贡献其他云厂商的Provider），自动化创建ecs并部署kubernetes集群并构建镜像，cloud build 是兼容性最好的构建方式， 基本可以100%的满足构建需求。缺点是需要创建按量计费的云主机会产生一定的成本。如果您要交付的环境涉及例如分布式存储这样的底层资源，建议使用此方式来进行构建。

```shell
# cloud模式拉起云服务器部署集群并发送上下文到云服务器中进行构建镜像并推送镜像，需要登录并创建build上下文目录和放入构建镜像所依赖的文件。
[root@sea ~]# sealer login registry.cn-qingdao.aliyuncs.com -u username -p password
[root@sea ~]# mkdir build && cd build && mv /root/recommended.yaml .
[root@sea build]# vi Kubefile
[root@sea build]# cat Kubefile
FROM kubernetes:v1.19.8
COPY recommended.yaml .
CMD kubectl apply -f recommended.yaml
[root@sea build]# ls
Kubefile  recommended.yaml
#执行构建
[root@sea build]# sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/my-cluster:v1.19.9 .
```

### 2.container build

> 与cloud build 原理类似，通过启动多个docker container作为kubernetes节点（模拟cloud模式的ECS）,从而启动一个kubernetes集群的方式来进行构建，可以消耗很少量的资源完成集群构建，缺点是不能很好的支持对底层资源依赖的场景。可以使用`-b container` 参数来指定build 类型 为container build 。

```shell
sealer build -b container -t my-cluster:v1.19.9 .
```

### 3.lite build

最轻量的构建方式， 原理是通过解析helm chart、提交镜像清单、解析manifest下的资源定义获取镜像清单并缓存，
配合Kubefile的定义，实现不用拉起kubernetes集群的轻量化构建，此种方式优点是资源消耗最低，有一台能够跑sealer的主机即可进行构建。缺点是无法覆盖一些场景，
例如无法获取通过operator部署的镜像，一些通过专有的管理工具进行交付的业务也无法解析获取到对应的镜像，lite build适用于已知镜像清单， 或者没有特殊的资源需求的场景。

Kubefile 示例：

```shell
FROM kubernetes:v1.19.8
COPY imageList manifests
COPY apollo charts
COPY helm /bin
CMD helm install charts/apollo
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

> 注意： 在lite build的场景下，因为build过程不会拉起集群，类似kubectl apply和helm install并不会实际执行成功， 但是会作为镜像的一层在交付集群的时候执行。

如上示例，lite构建会从如下三个位置解析会获取镜像清单，并将镜像缓存至registry：

* manifests/imageList: 内容就是镜像的清单，一行一个镜像地址。如果这个文件存在，则逐行提取镜像。imageList的文件名必须固定，不可更改，且必须放在manifests下。
* manifests 目录下的yaml文件: lite build将解析manifests目录下的所有yaml文件并从中提取镜像。
* charts 目录: helm chart应放置此目录下， lite build将通过helm引擎从helm chart中解析镜像地址。

lite build 操作示例，使用`-b lite` 参数来指定build 类型为 lite build。 假设Kubefile在当前目录下：

```shell
sealer build -b lite -t my-cluster:v1.19.9 .
```

构建完成将生成镜像：my-cluster:v1.19.9

## 私有仓库认证

在构建过程中，会存在使用私有仓库需要认证的场景， 在这个场景下， 进行镜像缓存时需要依赖docker的认证。可以在执行build操作前通过以下指令先进行login操作：

```shell
sealer login registry.com -u username -p password
```

另一个依赖场景，在交付完成后的，kubernetes node通过sealer内置的registry 代理到私有仓库且私有仓库需要认证时，可以通过自定义registry config来配置，sealer
优化和扩展了registry，使其可以同时支持多域名，多私有仓库的代理缓存。配置可参考: [registry配置文档](../user-guide/docker-image-cache.md)

可以通过定义Kubefile来自定义registry配置:

```shell
FROM kubernetes:v1.19.8
COPY registry_config.yaml etc/
```

## 自定义kubeadm配置

sealer将使用$Rootfs/etc目录下自定义配置文件来替换默认配置：

```yaml
   #自定义配置文件名称（文件名称固定）：
   bootstrapTemplateName       = "kubeadm-bootstrap.yaml.tmpl"
   initConfigTemplateName      = "kubeadm-init.yaml.tmpl"
   clusterConfigTemplateName   = "kubeadm-cluster-config.yaml.tmpl"
   kubeproxyConfigTemplateName = "kubeadm-kubeproxy-config.yaml.tmpl"
   kubeletConfigTemplateName   = "kubeadm-kubelet-config.yaml.tmpl"
   joinConfigTemplateName      = "kubeadm-join-config.yaml.tmpl"
```

### 例：自定义配置使用docker unix socket.

1. 自定义kubeadm init配置（文件名称需为kubeadm-init.yaml.tmpl）：

```yaml
apiVersion: {{.KubeadmAPI}}
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: {{.Master0}}
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

2. 自定义kubeadm join配置 （文件名称需为kubeadm-join-config.yaml.tmpl）：

```yaml
kind: JoinConfiguration
  { { - if .Master } }
controlPlane:
  localAPIEndpoint:
    advertiseAddress: {{.Master}}
    bindPort: 6443
  { { - end } }
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

3. 构建使用自定义配置重写默认配置的集群镜像：

```yaml
#Kubefile
FROM kubernetes:v1.19.8
COPY kubeadm-init.yaml.tmpl ./etc
COPY kubeadm-join-config.yaml.tmpl ./etc
```

> sealer build -b lite -t user-define-kubeadm-kubernetes:v1.19.8 .

### 默认模版配置文件内容：

> kubeadm-bootstrap.yaml.tmpl：
> ```yaml
> apiVersion: {{.KubeadmAPI}}
> caCertPath: /etc/kubernetes/pki/ca.crt
> discovery:
>   bootstrapToken:
>     {{- if .Master}}
>     apiServerEndpoint: {{.Master0}}:6443
>     {{else}}
>     apiServerEndpoint: {{.VIP}}:6443
>     {{end -}}
>     token: {{.TokenDiscovery}}
>     caCertHashes:
>     - {{.TokenDiscoveryCAHash}}
>     timeout: 5m0s
> ```
> kubeadm-init.yaml.tmpl：
> ```yaml
> apiVersion: {{.KubeadmAPI}}
> kind: InitConfiguration
> localAPIEndpoint:
>   advertiseAddress: {{.Master0}}
>   bindPort: 6443
> nodeRegistration:
>   criSocket: {{.CriSocket}}
> ```
> kubeadm-cluster-config.yaml.tmpl：
> ```yaml
> apiVersion: {{.KubeadmAPI}}
> kind: ClusterConfiguration
> kubernetesVersion: {{.Version}}
> controlPlaneEndpoint: "{{.ApiServer}}:6443"
> imageRepository: {{.Repo}}
> networking:
>   # dnsDomain: cluster.local
>   podSubnet: {{.PodCIDR}}
>   serviceSubnet: {{.SvcCIDR}}
> apiServer:
>   certSANs:
>   {{range .CertSANS -}}
>   - {{.}}
>   {{end -}}
>   extraArgs:
>     etcd-servers: {{.EtcdServers}}
>     feature-gates: TTLAfterFinished=true,EphemeralContainers=true
>     audit-policy-file: "/etc/kubernetes/audit-policy.yml"
>     audit-log-path: "/var/log/kubernetes/audit.log"
>     audit-log-format: json
>     audit-log-maxbackup: '"10"'
>     audit-log-maxsize: '"100"'
>     audit-log-maxage: '"7"'
>     enable-aggregator-routing: '"true"'
>   extraVolumes:
>   - name: "audit"
>     hostPath: "/etc/kubernetes"
>     mountPath: "/etc/kubernetes"
>     pathType: DirectoryOrCreate
>   - name: "audit-log"
>     hostPath: "/var/log/kubernetes"
>     mountPath: "/var/log/kubernetes"
>     pathType: DirectoryOrCreate
>   - name: localtime
>     hostPath: /etc/localtime
>     mountPath: /etc/localtime
>     readOnly: true
>     pathType: File
> controllerManager:
>   extraArgs:
>     feature-gates: TTLAfterFinished=true,EphemeralContainers=true
>     experimental-cluster-signing-duration: 876000h
>   extraVolumes:
>   - hostPath: /etc/localtime
>     mountPath: /etc/localtime
>     name: localtime
>     readOnly: true
>     pathType: File
> scheduler:
>   extraArgs:
>     feature-gates: TTLAfterFinished=true,EphemeralContainers=true
>   extraVolumes:
>   - hostPath: /etc/localtime
>     mountPath: /etc/localtime
>     name: localtime
>     readOnly: true
>     pathType: File
> etcd:
>   local:
>     extraArgs:
>       listen-metrics-urls: http://0.0.0.0:2381
> ```
> kubeadm-kubeproxy-config.yaml.tmpl：
> ```yaml
> apiVersion: kubeproxy.config.k8s.io/v1alpha1
> kind: KubeProxyConfiguration
> mode: "ipvs"
> ipvs:
>   excludeCIDRs:
>   - "{{.VIP}}/32"
> ```
> kubeadm-kubelet-config.yaml.tmpl：
> ```yaml
> apiVersion: kubelet.config.k8s.io/v1beta1
> kind: KubeletConfiguration
> authentication:
>   anonymous:
>     enabled: false
>   webhook:
>     cacheTTL: 2m0s
>     enabled: true
>   x509:
>     clientCAFile: /etc/kubernetes/pki/ca.crt
> authorization:
>   mode: Webhook
>   webhook:
>     cacheAuthorizedTTL: 5m0s
>     cacheUnauthorizedTTL: 30s
> cgroupDriver: {{ .CgroupDriver}}
> cgroupsPerQOS: true
> clusterDomain: cluster.local
> configMapAndSecretChangeDetectionStrategy: Watch
> containerLogMaxFiles: 5
> containerLogMaxSize: 10Mi
> contentType: application/vnd.kubernetes.protobuf
> cpuCFSQuota: true
> cpuCFSQuotaPeriod: 100ms
> cpuManagerPolicy: none
> cpuManagerReconcilePeriod: 10s
> enableControllerAttachDetach: true
> enableDebuggingHandlers: true
> enforceNodeAllocatable:
> - pods
> eventBurst: 10
> eventRecordQPS: 5
> evictionHard:
>   imagefs.available: 15%
>   memory.available: 100Mi
>   nodefs.available: 10%
>   nodefs.inodesFree: 5%
> evictionPressureTransitionPeriod: 5m0s
> failSwapOn: true
> fileCheckFrequency: 20s
> hairpinMode: promiscuous-bridge
> healthzBindAddress: 127.0.0.1
> healthzPort: 10248
> httpCheckFrequency: 20s
> imageGCHighThresholdPercent: 85
> imageGCLowThresholdPercent: 80
> imageMinimumGCAge: 2m0s
> iptablesDropBit: 15
> iptablesMasqueradeBit: 14
> kubeAPIBurst: 10
> kubeAPIQPS: 5
> makeIPTablesUtilChains: true
> maxOpenFiles: 1000000
> maxPods: 110
> nodeLeaseDurationSeconds: 40
> nodeStatusReportFrequency: 10s
> nodeStatusUpdateFrequency: 10s
> oomScoreAdj: -999
> podPidsLimit: -1
> port: 10250
> registryBurst: 10
> registryPullQPS: 5
> rotateCertificates: true
> runtimeRequestTimeout: 2m0s
> serializeImagePulls: true
> staticPodPath: /etc/kubernetes/manifests
> streamingConnectionIdleTimeout: 4h0m0s
> syncFrequency: 1m0s
> volumeStatsAggPeriod: 1m0s
> ```
> kubeadm-join-config.yaml.tmpl：
> ```yaml
> kind: JoinConfiguration
> {{- if .Master }}
> controlPlane:
>   localAPIEndpoint:
>     advertiseAddress: {{.Master}}
>     bindPort: 6443
> {{- end}}
> nodeRegistration:
>   criSocket: {{.CriSocket}}
> ```