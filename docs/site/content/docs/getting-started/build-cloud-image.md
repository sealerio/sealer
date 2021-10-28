+++
title = "Build cloud image"
description = "sealer build"
date = 2021-05-01T08:20:00+00:00
updated = 2021-05-01T08:20:00+00:00
draft = false
weight = 21
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Sealer can build images automatically by reading the instructions from a Kubefile. Using sealer build users can create an automated build that executes several command-line instructions in succession."
toc = true
top = false
+++

# Overview

A `Kubefile` is a text document that contains all the commands a user could call on the command line to assemble an
image.We can use the `Kubefile` to define a cluster image that can be shared and deployed offline. a `Kubefile` just
like `Dockerfile` which contains the build instructions to define the specific cluster.

## Kubefile instruction

### FROM instruction

The `FROM` instruction defines which base image you want reference, and the first instruction in Kubefile must be the
FROM instruction. Registry authentication information is required if the base image is a private image. By the way
official base images are available from the Sealer community.

> command format：FROM {your base image name}

USAGE：

For example ,use the base image `kubernetes:v1.19.9` which provided by the Sealer community to build a new cloud image.

`FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9`

### COPY instruction

The `COPY` instruction used to copy the contents from the context path such as file or directory to the `rootfs`. all
the cloud image is based on the [rootfs](../../../../api/cloudrootfs.md), and the default src path is
the `rootfs` .If the specified destination directory does not exist, sealer will create it automatically.

> command format：COPY {src dest}

USAGE：

For example , copy `mysql.yaml`to`rootfs`

`COPY mysql.yaml .`

### RUN instruction

The RUN instruction will execute any commands in a new layer on top of the current image and commit the results. The
resulting committed image will be used for the next step in the `Kubefile`.

> command format：RUN {command args ...}

USAGE：

For example ,Using `RUN` instruction to execute a commands that download kubernetes dashboard.

`RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml`

### CMD instruction

The format of CMD instruction is similar to RUN instruction, and also will execute any commands in a new layer. However,
the CMD command will be executed when the cluster is started . it is generally used to start applications or configure
the cluster. and it is different with `Dockerfile` CMD ,If you list more than one CMD in a `Kubefile` ,then all of them
will take effect.

> command format：CMD {command args ...}

USAGE：

For example ,Using `CMD` instruction to execute a commands that apply the kubernetes dashboard yaml.

`CMD kubectl apply -f recommended.yaml`

## Build command line

You can run the build command line after sealer installed. The current path is the context path ,default build type is
cloud and use build cache.

```shell
sealer build [flags] PATH
```

Flags:

```shell
Flags:
  -m, --mode string   requied,cluster image build type,default is cloud build.
  -t, --imageName string   requied,cluster image name.
  -f, --kubefile string    requied,kubefile filepath default is "Kubefile".
  --no-cache               build without cache.default is use cache to build.
  -h, --help               help for build.
```

### More Examples

### cloud build

`sealer build -f Kubefile -t my-kubernetes:1.19.9`

### container build

`sealer build -f Kubefile -t my-kubernetes:1.19.9 -m container`

### lite build

`sealer build -f Kubefile -t my-kubernetes:1.19.9 --mode lite`

## Build type

Currently, sealer build supports three build approaches for different requirement scenarios.

### 1.cloud build mode

The default build type. Based on cloud (currently only supported by Ali Cloud, welcome to contribute other cloud
providers), sealer can automatically create infra resources, deploy Kubernetes cluster and build images. And cloud Build
is the most compatible construction method and can basically meet the construction requirements of 100%. This build
approach is recommended if you are delivering a cloud image that involves infra resources such as persistence storage.
But the downside is that there is a cost associated with creating the cloud resources.

For example ,log in to the image registry, and create the build context directory,then put the files required for
building the image . In Cloud build, sealer will pull up the cluster and send the context to the cloud to build an image
,also will push the image automatically.

```shell
[root@sea ~]# sealer login registry.cn-qingdao.aliyuncs.com -u username -p password
[root@sea ~]# mkdir build && cd build && mv /root/recommended.yaml .
[root@sea build]# vi Kubefile
[root@sea build]# cat Kubefile
FROM kubernetes:v1.19.9
COPY recommended.yaml .
CMD kubectl apply -f recommended.yaml
[root@sea build]# ls
Kubefile  recommended.yaml
#start to build
[root@sea build]# sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/my-cluster:v1.19.9 .
```

### 2.container build mode

Similar to the cloud build mode, we can apply a Kubernetes cluster by starting multiple Docker containers as Kubernetes
nodes ( simulating cloud ECS), which consume very few resources to complete the build instruction. The disadvantage of
the container build is that some scenarios which rely on the infra resources is not supported very well.

You can specify the build type with the '-m container' argument to use container build.

```shell
sealer build -m container -t my-cluster:v1.19.9 .
```

### 3.lite build mode

The lightest build mode. By parsing the helm Chart, submitting the image list, parsing the kubernetes resource file
under the manifest to build the cloud image. and this can be done without starting the cluster

The advantages of this build mode is the lowest resource consumption . Any host installed sealer can use this mode to
build cloud image.

The disadvantage is that some scenarios cannot be covered. For example, the image deployed through the operator cannot
be obtained, and some images delivered through proprietary management tools are also can not be used.

In addition, some command such as `kubectl apply` or `helm install` will execute failed because the lite build process
will not pull up the cluster, but it will be saved as a layer of the image in the build stage.

Lite build is suitable for the scenarios where there is a list of known images or no special resource requirements.

Kubefile example：

```shell
FROM kubernetes:v1.19.8
COPY imageList manifests
COPY apollo charts
CMD helm install charts/apollo
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

As in the above example, the lite build will parse and cache the list of images to the registry from the following three
locations:

* `manifests/imageList`: The content is a list of images line by line, If this file exists, will be extracted to the
  desired image list . The file name of `imageList` must be fixed, unchangeable, and must be placed under manifests.

* `manifests` directory: Lite build will parse all the yaml files in the manifests directory and extract it to the
  desired image list.

* `charts` directory: this directory contains the helm chart, and lite build will resolve the image address from the
  helm chart through the helm engine.

You can specify the build type with the '-m lite' argument to use lite build.

```shell
sealer build -m lite -t my-cluster:v1.19.9 .
```

## Private registry

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
FROM kubernetes:v1.19.9
COPY registry_config.yaml etc/
```

## Custom kubeadm configuration

Sealer will replace the default configuration with a custom configuration file in $Rootfs/etc:

```yaml
   # custom config file name (file name is fixed)：
   bootstrapTemplateName       = "kubeadm-bootstrap.yaml.tmpl"
   initConfigTemplateName      = "kubeadm-init.yaml.tmpl"
   clusterConfigTemplateName   = "kubeadm-cluster-config.yaml.tmpl"
   kubeproxyConfigTemplateName = "kubeadm-kubeproxy-config.yaml.tmpl"
   kubeletConfigTemplateName   = "kubeadm-kubelet-config.yaml.tmpl"
   joinConfigTemplateName      = "kubeadm-join-config.yaml.tmpl"
```

### Example: Custom configuration using the Docker Unix socket.

1. define kubeadm init configuration (file name: kubeadm-init.yaml.tmpl) :

```yaml
apiVersion: {{.KubeadmAPI}}
kind: InitConfiguration
localAPIEndpoint:
  advertiseAddress: {{.Master0}}
  bindPort: 6443
nodeRegistration:
  criSocket: /var/run/dockershim.sock
```

2. define kubeadm join configuration (file name: kubeadm-join-config.yaml.tmpl) :

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

3. Build your own cloud image that override default configurations with custom configurations:

```yaml
#Kubefile
FROM kubernetes:v1.19.8
COPY kubeadm-init.yaml.tmpl ./etc
COPY kubeadm-join-config.yaml.tmpl ./etc
```

> sealer build -m lite -t user-define-kubeadm-kubernetes:v1.19.8 .

### Default template configuration file contents:

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