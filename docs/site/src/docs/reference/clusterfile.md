# Clusterfile

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