# 使用Clusterfile初始化集群

Clusterfile支持：用户自定义kubeadm，helm values 等配置的覆盖或合并，plugins 。。。

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    - key1=value1
    - key2=value2
    - key2=value3 #key2=[value2, value3]
  ssh:
    passwd:
    pk: xxx
    pkPasswd: xxx
    user: root
    port: "2222"
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ]
      env:
        - etcd-dir=/data/etcd
      ssh:
        user: xxx
        passwd: xxx
        port: "2222"
    - ips: [ 192.168.0.3 ]
      roles: [ node,db ]
```

## 使用案例

### 启动一个简单集群

3 masters and 1 node, It's so clearly and simple, cool

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
  ssh:
    passwd: xxx
  hosts:
    - ips: [ 192.168.0.2,192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
```

```shell script
sealer apply -f Clusterfile
```

### 重写ssh配置 (例如密码和port等)

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
  ssh:
    passwd: xxx
    port: "2222"
  hosts:
    - ips: [ 192.168.0.2 ] # 该master节点端口号与其他节点不同
      roles: [ master ]
      ssh:
        passwd: yyy
        port: "22"
    - ips: [ 192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
```

### 怎样设置自定义kubeadm配置

更好的方法是直接将 kubeadm 配置添加到 Clusterfile 中，当然每个集群镜像都有它的默认配置，您可以只定义这些配置的一部分，然后sealer将其合并到默认配置中。

```yaml
### 默认配置：
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
controlPlaneEndpoint: "apiserver.cluster.local:6443"
imageRepository: sea.hub:5000/library
networking:
  # dnsDomain: cluster.local
  podSubnet: 100.64.0.0/10
  serviceSubnet: 10.96.0.0/22
apiServer:
  certSANs:
    - 127.0.0.1
    - apiserver.cluster.local
    - 192.168.2.110
    - aliyun-inc.com
    - 10.0.0.2
    - 10.103.97.2
  extraArgs:
    etcd-servers: https://192.168.2.110:2379
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
    # advertiseAddress: 192.168.56.7
    bindPort: 6443
```

自定义kubeadm 配置（未指定参数使用默认值）

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
...
---
## 自定义配置必须指定kind类型
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
networking:
  podSubnet: 101.64.0.0/10
  serviceSubnet: 10.96.0.0/22
---
kind: KubeletConfiguration
authentication:
  webhook:
    cacheTTL: 2m1s
```

```shell
# 使用自定义kubeadm配置初始化集群
sealer apply -f Clusterfile
```

### 在config和脚本中使用env

在configs或yaml文件中使用env

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    - docker_dir=/var/lib/docker
    - ips=192.168.0.1;192.168.0.2;192.168.0.3 #ips=[192.168.0.1 192.168.0.2 192.168.0.3]
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ]
      env: # 不同节点支持覆盖env，数组使用分号隔开
        - docker_dir=/data/docker
        - ips=192.168.0.2;192.168.0.3
    - ips: [ 192.168.0.3 ]
      roles: [ node ]
```

在init.sh脚本中使用env:

```shell script
#!/bin/bash
echo $docker_dir ${ips[@]}
```

当sealer执行脚本时env的设置类似于：`docker_dir=/data/docker ips=(192.168.0.2;192.168.0.3) && sh init.sh`
该例子中, master ENV 是 `/data/docker`, node ENV 为 `/var/lib/docker`

### 支持Env渲染

本案例展示如何使用 env 设置dashboard服务目标端口

dashboard.yaml.tmpl:

```yaml
...
kind: Service
apiVersion: v1
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kubernetes-dashboard
spec:
  ports:
    - port: 443
      targetPort: {{ .DashBoardPort }}
  selector:
    k8s-app: kubernetes-dashboard
...
```

编写kubefile，此时需要将yaml复制到`manifests etc charts`目录下，sealer只渲染该目录下的文件：

sealer 将渲染 filename.yaml.tmpl 文件并创建一个名为 `filename.yaml` 的新文件

```yaml
FROM kubernetes:1.16.9
COPY dashobard.yaml.tmpl manifests/ # 仅支持`manifests etc charts` 目录下渲染文件
CMD kubectl apply -f manifests/dashobard.yaml
```

对于用户来说，只需要指定集群环境变量即可：

```shell script
sealer run -e DashBoardPort=8443 mydashboard:latest -m xxx -n xxx -p xxx
```

或者在Clusterfile中指定env

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: mydashobard:latest
  env:
    - DashBoardPort=8443
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ] # add role field to specify the node role
    - ips: [ 192.168.0.3 ]
      roles: [ node ]
```

### 使用env渲染Clusterfile

```shell
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    - podcidr=100.64.0.0/10
 ...
---
apiVersion: kubeadm.k8s.io/v1beta2
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
controlPlaneEndpoint: "apiserver.cluster.local:6443"
imageRepository: sea.hub:5000/library
networking:
  # dnsDomain: cluster.local
  podSubnet: {{ .podcidr }}
  serviceSubnet: 10.96.0.0/22
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
      # Configures Calico networking.
      calicoNetwork:
        # Note: The ipPools section cannot be modified post-install.
        ipPools:
        - blockSize: 26
          # Note: Must be the same as podCIDR
          cidr: {{ .podcidr }}
```

kubeadm和calico配置中的`{{ .podcidr }}`将被替换为Clusterfile.Env中的`podcidr`。