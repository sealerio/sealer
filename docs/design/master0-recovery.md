# Master0 recovery design

## Motivations

Master0 plays an important role in sealer cluster, it hosts many important components such as sealer builtin registry,
application images launched history and others cluster configuration files. if the master0 node is abnormal, a highly
available solution is required to recover it on other master nodes.

Currently, we have plans to save clusterfile to the cluster. while for built registry ,it is still deployed a single
instance on master0 node. So the following document are mainly for built registry.

## Proposal

### For builtin registry

* expose registry configuration to user thought clusterfile.
* distribute all registry data split from cluster image registry directory to each deploy host, whether it is a cluster
  image or an application image.
* save all registry configuration generated form sealer such as, basic auth information and https certificate.
* save the boot history of each application image information such as tag,version and image id to cluster, which was
  launched through the sealer run-app.

## Implementation

### For configuration of builtin registry

* expose below `Registry` struct to clusterfile which is contains configurations about local registry and remote
   registry.

```yaml
type Registry struct {
  Domain        string        `json:"domain,omitempty"`
  Port          int           `json:"port,omitempty"`
  Username      string        `json:"username,omitempty"`
  Password      string        `json:"password,omitempty"`
  LocalRegistry LocalRegistry `json:"local_registry,omitempty"`
}
  type LocalRegistry struct {
  // DeployHosts is the target host list that local registry will be deployed on.
  // if not set ,master0 will be the default value.
  DeployHosts []net.IP `json:"deploy_hosts,omitempty"`
  // InsecureMode indicated that whether the local registry is exposed in HTTPS.
  // if true sealer will generate default ssl cert.
  InsecureMode bool `json:"insecure_mode,omitempty,omitempty"`
}
```

* save registry configuration to cluster as configmap with name "sealer-registry" in namespace "kub-system".

```shell
# must exist, this is core registry configuration
/var/lib/sealer/data/my-cluster/rootfs/etc/registry.yml
#if local registry is not booted on insecure mode
/var/lib/sealer/data/my-cluster/rootfs/certs/sea.hub.crt
/var/lib/sealer/data/my-cluster/rootfs/certs/sea.hub.key
# if local registry is booted with basic auth.
/var/lib/sealer/data/my-cluster/rootfs/etc/registry_htpasswd
```

* save boot history of application image to cluster as configmap with name "sealer-apps" in namespace "kub-system".

```json
{
  "type": "kube-installer",
  "applications": [
    {
      "name": "nginx",
      "image_id": "sha256:6bc6bc3015103b5a20d25f80cc833c0df5c5fccfdcecacfedcea296c616a534e",
      "type": "helm",
      "launch_cmd": "helm install nginx application/apps/nginx/",
      "version": "v1"
    },
    {
      "name": "dashboard",
      "image_id": "sha256:6bc6bc3015103b5a20d25f80cc833c0df5c5fccfdcecacfedcea296c616a534e",
      "type": "kube",
      "launch_cmd": "kubectl apply -f application/apps/dashboard/",
      "version": "v1"
    }
  ],
  "launch": {
    "app_names": [
      "nginx",
      "dashboard"
    ]
  }
}
```

### For recovery of builtin registry

Step 1: Pre-distribute all registry data to all deploy hosts

* load registry configuration from cluster on new master0
* dump registry configuration files from cluster to rootfs on new master0
* launch builtin registry thought rootfs scripts and configuration files to recover registry service.
* replace the latest registry ip address on all cluster nodes.

Step 2: Post-distribute registry data on recovery host

* load registry configuration from cluster on new master0
* dump registry configuration files from cluster to rootfs on new master0
* distribute all registry data to recovery host thought boot history loaded from cluster on new master0
* launch builtin registry thought rootfs scripts and configuration files to recover registry service.
* replace the latest registry ip address on all cluster nodes.

Step 3: also need a sealer subcommand to do above recovery implementation

```shell
sealer alpha recover master0 --host 192.168.1.100
```