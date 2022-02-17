# Define your own CloudRootfs

All the files which run a kubernetes cluster needs.

Contains:

* Bin files, like docker, containerd, crictl ,kubeadm, kubectl...
* Config files, like kubelet systemd config, docker systemd config, docker daemon.json...
* Registry docker image.
* Some Metadata, like Kubernetes version.
* Registry files, contains all the docker image, like kubernetes core component docker images...
* Scripts, some shell script using to install docker and kubelet... sealer will call init.sh and clean.sh.
* Other static files

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
│   └── docker.tar.gz # cri bin files include docker,containerd,runc. 
├── etc
│   ├── 10-kubeadm.conf
│   ├── Clusterfile  # image default Clusterfile
│   ├── daemon.json # docker daemon config file. 
│   ├── docker.service 
│   ├── kubeadm.yml # kubeadm config including Cluster Configuration,JoinConfiguration and so on.
│   ├── kubelet.service
│   ├── registry_config.yml # docker registry config including storage root directory and http related config.
│   └── registry.yml # If the user wants to customize the username and password, can overwrite this file.
├── images
│   └── registry.tar  # registry docker image, will load this image and run a local registry in cluster
├── Kubefile
├── Metadata
├── README.md
├── registry # will mount this dir to local registry
│   └── docker
│       └── registry
├── scripts
│   ├── clean.sh
│   ├── docker.sh
│   ├── init-kube.sh
│   ├── init-registry.sh
│   ├── init.sh
│   └── kubelet-pre-start.sh
└── statics # yaml files, sealer will render values in those files
    └── audit-policy.yml
```

## How can I get CloudRootfs

1. Pull a BaseImage `sealer pull kubernetes:v1.19.8-alpine`
2. View the image layer information `sealer inspect kubernetes:v1.19.8-alpine`
3. Get into the BaseImage Layer `ls /var/lib/sealer/data/overlay2/{layer-id}`

You will find the CloudRootfs layer.

## Build your own CloudRootfs

You can edit any files in CloudRootfs you want, for example you want to define your own docker daemon.json, just edit it
and build a new CloudImage.

```shell script
FROM scratch
COPY . .
```

```shell script
sealer build -t user-defined-kubernetes:v1.19.8 .
```

Then you can use this image as a BaseImage.

## OverWrite CloudRootfs files

Sometimes you don't want to care about the CloudRootfs context, but need custom some config.

You can use `kubernetes:v1.19.8` as BaseImage, and use your own config file to overwrite the default file in
CloudRootfs.

For example: daemon.json is your docker engine config, using it to overwrite default config:

```shell script
FROM kubernetes:v1.19.8
COPY daemon.json etc/
```

```shell script
sealer build -t user-defined-kubernetes:v1.19.8 .
```

# Build your own cloud image

## Build with specific directory

#### images directory

Directory to save container images,the offline image in this directory will be load into the built-in registry when
sealer run.

Examples: copy offline tar file to this directory.

`COPY mysql.tar images`

#### plugin directory

Directory to save plugin files，the plugin file in this directory will be load into the runtime interface when sealer
run.

Examples: copy plugin config file to this directory.

plugin config: shell.yaml:

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

#### charts directory

Directory to save charts packages,When sealer builds, it parses the charts file in this directory, and downloads and
saves the corresponding container image.

Examples: copy nginx charts to this directory.

`COPY nginx charts`

#### manifests directory

Directory to save yaml file or "imageList" file,When sealer builds, it parses the yaml and "imageList" file in this
directory, and downloads and saves the corresponding container image.

Examples: copy "imageList" file to this directory.

```shell
[root@iZbp143f9driomgoqx2krlZ build]# cat imageList 
busybox
```

`COPY imageList manifests`

Examples: copy dashboard yaml file to this directory.

`COPY recommend.yaml manifests`

## Customize the private registry

Sealer optimizes and expands the docker registry, so that it can support proxy caching of multiple domain names and
multiple private registry at the same time.

During the build process, there will be a scenario where it uses a private registry which requires authentication. In
this scenario, the authentication of docker is required for image caching. You can perform the login operation first
through the following command before executing the build operation:

```shell
sealer login registry.com -u username -p password
```

Another dependent scenario， the kubernetes node is proxies to the private registry through the built-in registry of
sealer and the private registry needs to be authenticated, it can be configured through the custom registry config.Refer
to [registry config](../../../../user-guide/docker-image-cache.md)

You can customize the registry configuration by defining Kubefile:

```shell
FROM kubernetes:v1.19.8
COPY registry_config.yaml etc/
```

## Customize the kubeadm configuration

Sealer will replace the default configuration with a custom configuration file in $Rootfs/etc/kubeadm.yml.

### Example: Custom configuration using the Docker Unix socket.

1. customize kubeadm init configuration:

```yaml
apiVersion: kubeadm.k8s.io/v1beta2
kind: InitConfiguration
localAPIEndpoint:
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

2. customize kubeadm join configuration:

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

3. Build your own cloud image that override default configurations with custom configurations. Note that,the file name "
   kubeadm.yml" is fixed:

```yaml
#Kubefile
FROM kubernetes-clusterv2:v1.19.8
COPY kubeadm.yml etc
```

> sealer build -t user-define-kubeadm-kubernetes:v1.19.8 .

### Default kubeadm configuration file with completely contents:

pick any section of kubeadm.yml to customize:

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