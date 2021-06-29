# Get Started

## Build

### Build a helm CloudImage

1. Login your registry first
```shell script
sealer login registry.cn-qingdao.aliyuncs.com -u xxx -p xxx
```

2. Set you build context and Kubefile
> download bin files youself

```shell script
mkdir helm-context && cd helm-context
wget https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz
tar zxvf helm-v3.6.0-linux-amd64.tar.gz
```
Create a file Named Kubefile:
```shell script
FROM kubernetes:v1.19.9
COPY linux-amd64/helm /usr/bin
```

> OR using `RUN` command in Kubefile

If your Kubefile has `RUN` or `CMD` command, sealer will try to create a tmp Cluster to execute 
those commands.

So this case you need set cloud provider AK SK
```shell script
# like ALI_CLOUD ak sk
export ACCESSKEYID=LTAI5tx2dB2TgEkWAKU6wLfS
export ACCESSKEYSECRET=l7sHQ9vE1ZbxFxBkaKFb0YNSPOBt4D
```

Kubefile
```shell script
FROM kubernetes:v1.19.9
RUN wget https://get.helm.sh/helm-v3.6.0-linux-amd64.tar.gz && \
    tar zxvf helm-v3.6.0-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/bin
```

3. Build the CloudImage
```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-apps/helm:v3.6.0 .
```

## Share

Push CloudImage to a registry, full docker registry compatibility:
```shell script
sealer login registry.cn-qingdao.aliyuncs.com -u xxx -p
sealer push registry.cn-qingdao.aliyuncs.com/sealer-apps/kubernetes:v1.19.9 
sealer pull registry.cn-qingdao.aliyuncs.com/sealer-apps/kubernetes:v1.19.9 
```

We also can save the CloudImage as a tar file, copy and load it in your cluster.
```shell script
sealer save registry.cn-qingdao.aliyuncs.com/sealer-apps/kubernetes:v1.19.9 -o kubernetes.tar
sealer load -i kubernetes.tar
```

## Run

We can run a cluster suing `sealer run` or `sealer apply` command, `sealer apply` needs you edit a Clusterfile to tell
sealer the cluster configuration.

If you don't know how to write a Clusterfile, you can inspect a image to show the default Clusterfile:
```shell script
sealer inspect -c kubernetes:v1.19.9
```

### Run on exist servers

> Using sealer run

```shell script
sealer run kubernetes:v1.19.9 -m 192.168.0.2,192.168.0.3,192.168.0.4 -m 192.168.0.5,192.168.0.6,192.168.0.7 \
       -p xxxx # ssh passwd
```

Check the Cluster
```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                    STATUS ROLES AGE VERSION
izm5e42unzb79kod55hehvz Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r7z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r8z Ready master 18h v1.16.9
izm5ehdjw3kru84f0kq7r9z Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7raz Ready <none> 18h v1.16.9
izm5ehdjw3kru84f0kq7rbz Ready <none> 18h v1.16.9
```

> Using sealer apply, the provider should be BAREMETAL

Clusterfile:
```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: BAREMETAL
  ssh:
    # SSH login password, if you use the key to log in, you don’t need to set it
    passwd:
    ## The absolute path of the ssh private key file, for example /root/.ssh/id_rsa
    pk: xxx
    #  The password of the ssh private key file, if there is none, set it to ""
    pkPasswd: xxx
    # ssh login user
    user: root
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    ipList:
     - 172.20.126.4
     - 172.20.126.5
     - 172.20.126.6
  nodes:
    ipList:
     - 172.20.126.8
     - 172.20.126.9
     - 172.20.126.10
  ## Optional, do not enter or enter "", the first IP of masters will be used
  registry: 172.20.126.5
```
```shell script
sealer apply -f Clusterfile
```

> scale up and down 

you just need to add or delete ip in masters or nodes ipList and reapply.

OR using join command to scale up.

```shell script
sealer join --masters 192.168.0.2 --nodes 192.168.0.3
```

### Run on Cloud

Set the Cloud provider AK SK before you install a Cluster, Now support ALI_CLOUD.

```shell script
export ACCESSKEYID=xxx
export ACCESSKEYSECRET=xxx
```

> Using sealer run

You just need specify the machine(VM) resource configuration and counts.

`sealer run kubernetes:v1.19.9 -m 1 -n 1`

> Using sealer apply, the provider should be ALI_CLOUD

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  provider: ALI_CLOUD
  network:
    # in use NIC name
    interface: eth0
    # Network plug-in name
    cniName: calico
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
    withoutCNI: false
  certSANS:
    - aliyun-inc.com
    - 10.0.0.2
    
  masters:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
  nodes:
    cpu: 4
    memory: 4
    count: 3
    systemDisk: 100
    dataDisks:
    - 100
```

```shell script
sealer apply -f Clusterfile
```

> scale up and down

just edit `.sealer/my-cluster/Clusterfile` set masters.count or nodes.count to you desired number and reapply:
```shell script
sealer apply -f .sealer/my-cluster/Clusterfile
```

### Clean 

cluster-name is defined in metadata.name
```shell script
sealer delete -f .sealer/[cluster-name]/Clusterfile
```
if you using cloud mod, sealer will delete the infa resouce too.

