# 构建集群镜像

集群镜像的构建过程与构建Docker镜像很类似，通过定义一个Kubefile文件，文件中定义一些指令，build结束就可以把集群启动依赖的所有文件和docker镜像打包成集群镜像了。

## 构建模式

目前sealer支持三种构建模式：

* 默认模式，lite build, 这种方式构建过程中不需要启动一个k8s集群，通过解析用户提供的yaml文件或者helm chart或者用户自定义的imageList来拉取集群中依赖的容易镜像
* Cloud build，这种模式会在云上启动一个kubernetes集群，并在集群中执行Kubefile中的指令，依赖公有云，需要配置AK SK，好处是能发现CRD里面依赖的容器镜像。
* container build, 这种模式也会起一个kubernetes集群，不过是通过docker模拟了虚拟机节点，可以在本地直接构建。

### lite build 模式

这是默认的构建模式，比较轻量快速，会解析用户的yaml文件或者helm chart 分析出里面包含的容器镜像，并拉取下来存储到集群镜像自带的registry中

Kubefile:

```shell
FROM kubernetes:v1.19.8
COPY imageList manifests
COPY apollo charts
COPY helm /bin
CMD helm install charts/apollo
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

* `manifests` 目录: 这是一个特殊的目录，sealer build的时候会解析这个目录下面的所有yaml文件并把里面的容器镜像地址提取出来，然后拉取。用户标准的kubernetes yaml不放在这个目录的话不会处理。
* `charts` 目录: 这也是一个特殊目录，sealer会执行helm template的能力，然后提取chart中的容器镜像地址，拉取并存储到集群镜像中。chart不拷贝到这个目录下不处理。
* `manifests/imageList`: 这是个特殊的文件里面是其它需要拉取的镜像地址列表，比如镜像地址在CRD中sealer解析不到，那就需要手动配置到这个文件中。

imageList 内容示例：

```
nginx:latest
mysql:5.6
```

Build集群镜像：

```shell
sealer build -t my-cluster:v1.19.9 .
```

### Cloud build 模式

Cloud build的模式不会要求yaml文件或者helm chart等放的具体位置，因为会真的创建一个集群就可以在集群内获取到集群中依赖的容器镜像信息，所以可以直接Build:

```shell script
sealer build -m cloud -t my-cluster:v1.19.9 .
```

container build，无须指定AK SK也可build:

```shell script
sealer build -m container -t my-cluster:v1.19.8 .
```

## 私有镜像仓库

集群镜像也可以被推送到docker registry中

```shell
sealer login registry.cn-qingdao.aliyuncs.com -u username -p password
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/kuberentes:v1.19.8
sealer pull registry.cn-qingdao.aliyuncs.com/sealer-io/kuberentes:v1.19.8
```

集群镜像中自带一个docker registry, 所有容器镜像会存储在这个registry中，可以自定义该registry的一些配置：

[registry config](../../../../design/docker-image-cache.md)

编辑好 `registry_config.yaml`配置文件，然后在Kubefile中进行overwrite:

```shell
FROM kubernetes:v1.19.8
COPY registry_config.yaml etc/
```

## 自定义 kubeadm 配置

Sealer 会把用户自定义的kubeadm配置文件与默认文件merge, 集群镜像中 $Rootfs/etc/kubeadm.yml 文件为镜像默认的Kubeadm配置，

用户可以直接在镜像中覆盖它，或者在Clusterfile中定义kubeadm配置文件，执行时会把Clusterfile中的kubeadm配置与镜像中的合并。

这里合并时只覆盖对应字段，而不是全部替换，比如你只关心 `bindPort` 这一个参数，那只需要在Clusterfile中配置这一个字段即可，不用写全量配置。

### 如自定义 Docker Unix socket.

1. 修改 kubeadm init configuration:

```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
localAPIEndpoint:
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

2. 修改 kubeadm join configuration:

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

3. 把该文件命名为kubeadm.yml, 构建时拷贝到etc/下即可

```yaml
#Kubefile
FROM kubernetes-clusterv2:v1.19.8
COPY kubeadm.yml etc
```

这里注意kubeadm.yml中也不需要全量配置，只需要配置对应关心的字段，sealer会与默认配置合并。

merge规则：Clusterfile中的配置 > 集群镜像中的配置 > 默认kubeadm配置(硬编码在代码中)

> sealer build -t user-define-kubeadm-kubernetes:v1.19.8 .

## 默认kubeadm 配置文件全内容:

kubeadm.yml：

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
