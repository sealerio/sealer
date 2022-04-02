# Using Clusterfile to init a cluster

Clusterfile support more configs like user defined kubeadm config, helm values config overwrite, plugins ...

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
  env:
    - key1=value1
    - key2=value2;value3 #key2=[value2, value3]
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

### Overwrite ssh config (for example password,and port)

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
    - ips: [ 192.168.0.2 ] # this master ssh port is different with others.
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

The better way is to add kubeadm config directly into Clusterfile, of course every CloudImage has it default config:
You can only define part of those configs, sealer will merge then into default config.

```yaml
### default kubeadm config:
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

Custom kubeadm configuration (use default configuration for other parts)

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.8
...
---
## Custom configurations must specify kind
kind: ClusterConfiguration
kubernetesVersion: v1.19.8
networking:
  # dnsDomain: cluster.local
  podSubnet: 101.64.0.0/10
  serviceSubnet: 10.96.0.0/22
---
## Custom configurations must specify kind
kind: KubeletConfiguration
authentication:
  webhook:
    cacheTTL: 2m1s
```

```shell
# Initialize the cluster using custom kubeadm
sealer apply -f Clusterfile
```

### Using ENV in configs and script

Using ENV in configs or yaml files

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
      roles: [ master ] # add role field to specify the node role
      env: # overwrite some nodes has different env config, arrays are separated by semicolons
        - docker_dir=/data/docker
        - ips=192.168.0.2;192.168.0.3
    - ips: [ 192.168.0.3 ]
      roles: [ node ]
```

Using ENV in init.sh script:

```shell script
#!/bin/bash
echo $docker_dir
```

When sealer run the script will set ENV like this: `docker_dir=/data/docker && sh init.sh`
In this case, master ENV is `/data/docker`, node ENV is by default `/var/lib/docker`

### Env render support

support [sprig](http://masterminds.github.io/sprig/) template functions.
This case show you how to use env to set dashboard service target port

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

To write kubefile, you need to copy yaml to the "manifests" directory at this time, sealer only renders the files in
this directory:

sealer will render the .tmpl file and create a new file named `dashboard.yaml`

```yaml
FROM kubernetes:1.16.9
COPY dashobard.yaml.tmpl manifests/ # only support render template files in `manifests etc charts` dirs
CMD kubectl apply -f manifests/dashobard.yaml
```

For users, they only need to specify the cluster environment variables:

```shell script
sealer run -e DashBoardPort=8443 mydashboard:latest -m xxx -n xxx -p xxx
```

Or set env in Clusterfile

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

### Render env in Clusterfile

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

Replace `podcidr` in kubeadm and Calico configurations with `podcidr` in Env in Clusterfile.

### Overwrite CMD support

This case show you how to use `cmd` fields of Clusterfile to overwrite cloud image startup.

Kubefile:

```shell
FROM kubernetes:v1.19.8
CMD [kubectl apply -f mysql, kubectl apply -f redis, kubectl apply -f saas]
```

If user wants to overwrite the default startup ,they only need to specify the `cmd` fields of Clusterfile.In this
case,will only start `kubectl apply -f redis` and `kubectl apply -f saas`.

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: myapp:latest
  cmd:
    - kubectl apply -f redis
    - kubectl apply -f saas
  hosts:
    - ips: [ 192.168.0.2 ]
      roles: [ master ] # add role field to specify the node role
    - ips: [ 192.168.0.3 ]
      roles: [ node ]
```
