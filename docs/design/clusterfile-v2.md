# Clusterfile v2 design

## Motivations

Clusterfile v1 not match some requirement.

* Different node has different ssh config like passwd.
* Not clear, confused about what argument should put into Clusterfile.
* Coupling the infra config and cluster config.

## Proposal

* Delete provider field
* Add env field
* Modify hosts field, add ssh and env rewrite
* Delete all kubeadm config

```yaml
apiVersion: sealer.io/v2
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
      roles: [ master ] # add role field to specify the node role
      env: # rewrite some nodes has different env config
        - etcd-dir=/data/etcd
      ssh: # rewrite ssh config if some node has different passwd...
        user: xxx
        passwd: xxx
        port: "2222"
    - ips: [ 192.168.0.3 ]
      roles: [ node,db ]
```

## Use cases

### Apply a simple cluster by default

3 masters and a node, It's so clearly and simple, cool

```yaml
apiVersion: sealer.io/v2
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

### Overwrite ssh config (for example password,and port)

```yaml
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
  ssh:
    passwd: xxx
    port: "2222"
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ]
      ssh:
        passwd: yyy
        port: "22"
    - ips: [ 192.168.0.3,192.168.0.4 ]
      roles: [ master ]
    - ips: [ 192.168.0.5 ]
      roles: [ node ]
```

### How to define your own kubeadm config

The better way is to add kubeadm config directly into Clusterfile, of course every ClusterImage has it default config:
You can only define part of those configs, sealer will merge then into default config.

```yaml
apiVersion: kubeadm.k8s.io/v1beta3
kind: InitConfiguration
localAPIEndpoint:
  # advertiseAddress: 192.168.2.110
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock

---
apiVersion: kubeadm.k8s.io/v1beta3
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
controlPlaneEndpoint: "apiserver.cluster.local:6443"
imageRepository: sea.hub:5000
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
apiVersion: kubeadm.k8s.io/v1beta3
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

### Using Kubeconfig to overwrite kubeadm configs

If you don't want to care about so much Kubeadm configs, you can use `KubeConfig` object to overwrite(json patch merge) some fields.

```yaml
apiVersion: sealer.io/v2
kind: KubeadmConfig
metadata:
  name: default-kubernetes-config
spec:
  localAPIEndpoint:
    advertiseAddress: 192.168.2.110
    bindPort: 6443
  nodeRegistration:
    criSocket: /var/run/dockershim.sock
  kubernetesVersion: v1.19.8
  controlPlaneEndpoint: "apiserver.cluster.local:6443"
  imageRepository: sea.hub:5000
  networking:
    podSubnet: 100.64.0.0/10
    serviceSubnet: 10.96.0.0/22
  apiServer:
    certSANs:
      - sealer.cloud
      - 127.0.0.1
  clusterDomain: cluster.local
```

### Using ENV in configs and script

Using ENV in configs or yaml files [check this](https://github.com/sealerio/sealer/blob/main/docs/design/global-config.md#global-configuration)

```yaml
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    docker-dir: /var/lib/docker
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ] # add role field to specify the node role
      env: # overwrite some nodes has different env config
        docker-dir: /data/docker
    - ips: [ 192.168.0.3 ]
      roles: [ node ]
```

Using ENV in init.sh script:

```shell script
#!/bin/bash
echo $docker-dir
```

When sealer run the script will set ENV like this: `docker-dir=/data/docker && sh init.sh`
In this case, master ENV is `/data/docker`, node ENV is by default `/var/lib/docker`

### How to use cloud infra

If you're using public cloud, you needn't to config the ip field in Cluster Object. The infra Object will tell sealer to
apply resource from public cloud, then render the ip list to Cluster Object.

```yaml
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
---
apiVersion: sealer.io/v2
kind: Infra
metadata:
  name: alicloud
spec:
  provider: ALI_CLOUD
  ssh:
    passwd: xxx
    port: "2222"
  hosts:
    - count: 3
      role: [ master ]
      cpu: 4
      memory: 4
      systemDisk: 100
      dataDisk: [ 100,200 ]
    - count: 3
      role: [ node ]
      cpu: 4
      memory: 4
      systemDisk: 100
      dataDisk: [ 100, 200 ]
```

After `sealer apply -f Clusterfile`, The cluster object will update:

```yaml
apiVersion: sealer.io/v2
kind: Cluster
metadata:
  name: default-kubernetes-cluster
spec:
  image: kubernetes:v1.19.8
  ssh:
    passwd: xxx
    port: "2222"
  hosts:
    - ips: [ 192.168.0.3 ]
      roles: [ master ]
...
```

### Env render support

[Env render](https://github.com/sealerio/sealer/blob/main/docs/design/global-config.md#global-configuration)