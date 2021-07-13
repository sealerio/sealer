# Build a kubernetes-withcalico CloudImage

```shell script
sealer build -t kubernetes-withcalico:v1.19.9 .
sealer push kubernetes-withcalico:v1.19.9
```

# Using kubernetes-withcalico CloudImage  

This image contains the default Calico configuration [custom-resources.yaml](latest/etc/custom-resources.yaml). 

Clusterfile:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes-withcalico:v1.19.9
  network:
    # default Calico configuration use this cidr
    podCIDR: 100.64.0.0/10 
...

```

```shell script
sealer apply -f Clusterfile
```

If you want to override the default Calico configuration file, you need to add sealer configuration to the Clusterfile. 

Clusterfile:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes-withcalico:v1.19.9
  network:
    podCIDR: 100.64.0.0/10    
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
      # Configures Calico networking.
      calicoNetwork:
        # Note: The ipPools section cannot be modified post-install.
        ipPools:
        - blockSize: 26
          # Note: Must be the same as podCIDR
          cidr: 100.64.0.0/10
          encapsulation: IPIP
          natOutgoing: Enabled
          nodeSelector: all()
        nodeAddressAutodetectionV4:
          interface: "eth.*|en.*"
```

```shell script
sealer apply -f Clusterfile
```

For more information about calico installation configuration, see [the installation reference](https://docs.projectcalico.org/reference/installation/api#operator.tigera.io/v1.Installation).

# Using kubernetes-withcalico CloudImage as Base Image

```shell script
FROM kubernetes-withcalico:v1.19.9
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```

