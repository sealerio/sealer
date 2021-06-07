# sealer release v0.1.5 test report

# Cloud mode:

##   init:

```shell
1.wget https://github.com/alibaba/sealer/releases/download/v0.1.5-rc/sealer-0.0.0-linux-amd64.tar.gz && tar zxvf sealer-0.0.0-linux-amd64.tar.gz && mv sealer /usr/local/bin/
2.export 'ACCESSKEYID'='***'&&export 'ACCESSKEYSECRET'='***'&&export 'RegionID'='***'
3.sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 //（Cluster started successfully）
```

The default password is the Clusterfile password in the Run image.

**Check the Clusterfile：**

```shell
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# cat /root/.sealer/my-cluster/Clusterfile
apiVersion: zlink.aliyun.com/v1alpha1
kind: Cluster
metadata:
  annotations:
    sea.aliyun.com/ClusterEIP: 47.108.132.245
    sea.aliyun.com/EipID: eip-2vca3x1lxu96ropt66am5
    sea.aliyun.com/Master0ID: i-2vc2ce4hs85f7ql6dqda
    sea.aliyun.com/Master0InternalIP: 172.16.0.194
    sea.aliyun.com/MasterIDs: i-2vc2ce4hs85f7ql6dqdb,i-2vc2ce4hs85f7ql6dqdc,i-2vc2ce4hs85f7ql6dqda
    sea.aliyun.com/NodeIDs: i-2vc4eamks4zejf9n16tm,i-2vc4eamks4zejf9n16tn,i-2vc4eamks4zejf9n16tl
    sea.aliyun.com/RegionID: cn-chengdu
    sea.aliyun.com/SecurityGroupID: sg-2vcdx512qke5iqpx1iwx
    sea.aliyun.com/VSwitchID: vsw-2vc55ufqm05qb8zdxux0h
    sea.aliyun.com/VpcID: vpc-2vcj16rdif2upvgeowdu6
    sea.aliyun.com/ZoneID: cn-chengdu-a
  creationTimestamp: null
  name: my-cluster
spec:
  certSANS:
  - aliyun-inc.com
  - 10.0.0.2
  image: kubernetes:v1.19.9
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.194
    - 172.16.0.196
    - 172.16.0.195
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.198
    - 172.16.0.199
    - 172.16.0.197
    memory: "4"
    systemDisk: "100"
  provider: ALI_CLOUD
  ssh:
    passwd: Seadent123
    pk: xxx
    pkPasswd: xxx
    user: root
status: {}
```





## scale up:

#### 1. Modify /root/sealer/mycluster/Clusterfile file Masters, Nodes Count number：

```shell
  masters:
    count: "4"//Master adds one node
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.194
    - 172.16.0.196
    - 172.16.0.195
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "4"//node adds one node
```

#### 2. Execute Sealer apply-f /root/.sealer/my-cluster/Clusterfile (Successfully executed):



```shell
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# kubectl get nodes
NAME                      STATUS   ROLES    AGE     VERSION
iz2vc1hagqsoc9ut8fp511z   Ready    <none>   2m50s   v1.19.9
iz2vc2ce4hs85f7ql6dqdaz   Ready    master   22m     v1.19.9
iz2vc2ce4hs85f7ql6dqdbz   Ready    master   20m     v1.19.9
iz2vc2ce4hs85f7ql6dqdcz   Ready    master   21m     v1.19.9
iz2vc4eamks4zejf9n16tlz   Ready    <none>   19m     v1.19.9
iz2vc4eamks4zejf9n16tmz   Ready    <none>   19m     v1.19.9
iz2vc4eamks4zejf9n16tnz   Ready    <none>   19m     v1.19.9
iz2vcdx512qke5j0l5bksrz   Ready    master   3m6s    v1.19.9
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# cat /root/.sealer/my-cluster/Clusterfile
apiVersion: zlink.aliyun.com/v1alpha1
kind: Cluster
metadata:
  annotations:
    sea.aliyun.com/ClusterEIP: 47.108.132.245
    sea.aliyun.com/EipID: eip-2vca3x1lxu96ropt66am5
    sea.aliyun.com/Master0ID: i-2vc2ce4hs85f7ql6dqda
    sea.aliyun.com/Master0InternalIP: 172.16.0.194
    sea.aliyun.com/MasterIDs: i-2vc2ce4hs85f7ql6dqdb,i-2vc2ce4hs85f7ql6dqdc,i-2vc2ce4hs85f7ql6dqdai-2vcdx512qke5j0l5bksr
    sea.aliyun.com/NodeIDs: i-2vc4eamks4zejf9n16tm,i-2vc4eamks4zejf9n16tn,i-2vc4eamks4zejf9n16tli-2vc1hagqsoc9ut8fp511
    sea.aliyun.com/RegionID: cn-chengdu
    sea.aliyun.com/SecurityGroupID: sg-2vcdx512qke5iqpx1iwx
    sea.aliyun.com/VSwitchID: vsw-2vc55ufqm05qb8zdxux0h
    sea.aliyun.com/VpcID: vpc-2vcj16rdif2upvgeowdu6
    sea.aliyun.com/ZoneID: cn-chengdu-a
  creationTimestamp: null
  name: my-cluster
spec:
  certSANS:
  - aliyun-inc.com
  - 10.0.0.2
  image: kubernetes:v1.19.9
  masters:
    count: "4"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.194
    - 172.16.0.196
    - 172.16.0.195
    - 172.16.0.200
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "4"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.198
    - 172.16.0.199
    - 172.16.0.197
    - 172.16.0.201
    memory: "4"
    systemDisk: "100"
  provider: ALI_CLOUD
  ssh:
    passwd: Seadent123
    pk: xxx
    pkPasswd: xxx
    user: root
status: {}
```

## scale down:

#### 1. Modify /root/sealer/mycluster/Clusterfile file Masters, Nodes Count number:

```shell
  masters:
    count: "3"//Masters reduced by one
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.194
    - 172.16.0.196
    - 172.16.0.195//master iplist reduced by one
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"//nodes reduced by one
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.198
    - 172.16.0.199
    - 172.16.0.197//node iplist reduced by one
    memory: "4"
    systemDisk: "100"
```

#### 2. Execute sealer apply -f /root/.sealer/my-cluster/Clusterfile（Successfully executed）:



```shell
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# sealer apply -f /root/.sealer/my-cluster/Clusterfile
2021-06-03 15:03:50 [INFO] reconcile master instances success [172.16.0.194 172.16.0.196 172.16.0.195]
2021-06-03 15:03:51 [INFO] reconcile node instances success [172.16.0.198 172.16.0.199 172.16.0.197]
2021-06-03 15:03:51 [INFO] desired master 3, current master 4, desired nodes 3, current nodes 4
2021-06-03 15:03:51 [INFO] delete nodes [172.16.0.200 172.16.0.201]
2021-06-03 15:03:51 [INFO] scale the cluster success
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# kubectl get nodes
NAME                      STATUS   ROLES    AGE   VERSION
iz2vc2ce4hs85f7ql6dqdaz   Ready    master   39m   v1.19.9
iz2vc2ce4hs85f7ql6dqdbz   Ready    master   37m   v1.19.9
iz2vc2ce4hs85f7ql6dqdcz   Ready    master   38m   v1.19.9
iz2vc4eamks4zejf9n16tlz   Ready    <none>   37m   v1.19.9
iz2vc4eamks4zejf9n16tmz   Ready    <none>   37m   v1.19.9
iz2vc4eamks4zejf9n16tnz   Ready    <none>   37m   v1.19.9
```

#### 3. Check the Clusterfile

```shell
[root@iZ2vc2ce4hs85f7asxhaipZ ~]# cat /root/.sealer/my-cluster/Clusterfile
apiVersion: zlink.aliyun.com/v1alpha1
kind: Cluster
metadata:
  annotations:
    ShouldBeDeleteInstancesIDs: ""
    sea.aliyun.com/ClusterEIP: 47.108.132.245
    sea.aliyun.com/EipID: eip-2vca3x1lxu96ropt66am5
    sea.aliyun.com/Master0ID: i-2vc2ce4hs85f7ql6dqda
    sea.aliyun.com/Master0InternalIP: 172.16.0.194
    sea.aliyun.com/MasterIDs: i-2vc2ce4hs85f7ql6dqdb,i-2vc2ce4hs85f7ql6dqdc,i-2vc2ce4hs85f7ql6dqdai-2vcdx512qke5j0l5bksr
    sea.aliyun.com/NodeIDs: i-2vc4eamks4zejf9n16tm,i-2vc4eamks4zejf9n16tn,i-2vc4eamks4zejf9n16tli-2vc1hagqsoc9ut8fp511
    sea.aliyun.com/RegionID: cn-chengdu
    sea.aliyun.com/SecurityGroupID: sg-2vcdx512qke5iqpx1iwx
    sea.aliyun.com/VSwitchID: vsw-2vc55ufqm05qb8zdxux0h
    sea.aliyun.com/VpcID: vpc-2vcj16rdif2upvgeowdu6
    sea.aliyun.com/ZoneID: cn-chengdu-a
  creationTimestamp: null
  name: my-cluster
spec:
  certSANS:
  - aliyun-inc.com
  - 10.0.0.2
  image: kubernetes:v1.19.9
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.194
    - 172.16.0.196
    - 172.16.0.195
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.198
    - 172.16.0.199
    - 172.16.0.197
    memory: "4"
    systemDisk: "100"
  provider: ALI_CLOUD
  ssh:
    passwd: Seadent123
    pk: xxx
    pkPasswd: xxx
    user: root
status: {}
```

#### 4. Scale down node machine successfully deleted：


![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2021/png/7656565/1622789105210-ce7d74b4-9e97-4a0e-bd5a-0c5a66a4d80f.png)

##   

#  

# Bare metal mode：

# A.Executed on master0：   4C4G centos7.9

## 1. init：

```shell
1.wget https://github.com/alibaba/sealer/releases/download/v0.1.5-rc/sealer-0.0.0-linux-amd64.tar.g && tar zxvf sealer-0.0.0-linux-amd64.tar.gz && mv sealer /usr/local/bin/

2.sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters 172.16.0.165,172.16.0.166,172.16.0.167 --nodes 172.16.0.168,172.16.0.169 --passwd Sealer123
 集群成功启动
[root@test-baremetal004 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE   VERSION
test-baremetal003   Ready    master   12m   v1.19.9
test-baremetal004   Ready    master   13m   v1.19.9
test-baremetal005   Ready    master   12m   v1.19.9
test-baremetal006   Ready    <none>   11m   v1.19.9
test-baremetal007   Ready    <none>   11m   v1.19.9
```



**Problem: The number of counts in /root/.sealer/my-cluster/Clusterfile does not change but does not affect the result**

**check Clusterfile：**

```shell
[root@test-baremetal004 ~]# cat /root/.sealer/my-cluster/Clusterfile
apiVersion: zlink.aliyun.com/v1alpha1
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  certSANS:
  - aliyun-inc.com
  - 10.0.0.2
  image: registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.165
    - 172.16.0.166
    - 172.16.0.167
~
  nodes:
    count: "3" //Count does not match the number of Iplists
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.168
    - 172.16.0.169
~
```

## 2. Scale up

**Add node Masters: 172.16.0.170, 172.16.0.171;  nodes: 172.16.0.172**

1. **Modify Clusterfile：**

```shell
~
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.165
    - 172.16.0.166
    - 172.16.0.167
    - 172.16.0.170//masters adds one node
    - 172.16.0.171
~
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.168
    - 172.16.0.169
    - 172.16.0.172//nodes adds one node
```

1. **Execute sealer apply -f /root/.sealer/my-cluster/Clusterfile （executed successfully)：**

```shell
[root@test-baremetal004 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE   VERSION
test-baremetal001   Ready    <none>   33s   v1.19.9
test-baremetal002   Ready    master   56s   v1.19.9
test-baremetal003   Ready    master   26m   v1.19.9
test-baremetal004   Ready    master   27m   v1.19.9
test-baremetal005   Ready    master   25m   v1.19.9
test-baremetal006   Ready    <none>   25m   v1.19.9
test-baremetal007   Ready    <none>   25m   v1.19.9
test-baremetal009   Ready    master   74s   v1.19.9
```

## 3.  Scale down

1. **Modify Clusterfile**：

```shell
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.165
    - 172.16.0.166
    - 172.16.0.167 //master reduced by one
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.168
    - 172.16.0.169 //node reduced by one
```

2. **Execute sealer apply -f /root/.sealer/my-cluster/Clusterfile（executed successfully）：**

```shell
[root@test-baremetal004 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE   VERSION
test-baremetal003   Ready    master   29m   v1.19.9
test-baremetal004   Ready    master   29m   v1.19.9
test-baremetal005   Ready    master   28m   v1.19.9
test-baremetal006   Ready    <none>   27m   v1.19.9
test-baremetal007   Ready    <none>   27m   v1.19.9
```

**Problem：**

**Execution Sealer delete-f Clusterfile directly after Scale Down. The two nodes deleted by Scale Down will still have the rootfs directory and the Seatutil tool **

```shell
[root@baremetal002 ~]# ssh root@172.16.0.170
root@172.16.0.160's password:
Last login: Thu Jun  3 17:08:30 2021 from 172.16.0.156
Welcome to Alibaba Cloud Elastic Compute Service !

[root@iZ2vcdx512qke5k9yuwucfZ ~]# ls /var/lib/sealer/data/my-cluster/rootfs/
bin  cri  images                    Kubefile  README.md  scripts  TREE.md
cni  etc  kubeadm-join-config.yaml  Metadata  registry   statics
```



## 4. Delete

**Execute Sealer delete-f /root/.Sealer /my-cluster/Clusterfile (the last step reports an error, does not affect the result) :**

**All nodes were deleted successfully**

```shell
/bin/sh: /var/lib/sealer/data/my-cluster/rootfs/scripts/clean.sh: No such file or directory
2021-06-04 15:53:54 [EROR] exec command failed Process exited with status 127
2021-06-04 15:53:54 [EROR] 172.16.0.165:exec /bin/sh -c /var/lib/sealer/data/my-cluster/rootfs/scripts/clean.sh failed, exec command failed 172.16.0.165 /bin/sh -c /var/lib/sealer/data/my-cluster/rootfs/scripts/clean.sh
2021-06-04 15:53:55 [INFO] [ssh][172.16.0.169] : rm -rf /var/lib/sealer/data/my-cluster/rootfs
2021-06-04 15:53:55 [INFO] [ssh][172.16.0.168] : rm -rf /var/lib/sealer/data/my-cluster/rootfs
2021-06-04 15:53:55 [INFO] [ssh][172.16.0.167] : rm -rf /var/lib/sealer/data/my-cluster/rootfs
2021-06-04 15:53:55 [INFO] [ssh][172.16.0.166] : rm -rf /var/lib/sealer/data/my-cluster/rootfs
2021-06-04 15:53:55 [EROR] unmountRootfs failed
```

**Error: /var/lib/sealer/data/my-cluster directory does not exist **

**// Seautil tool still exists after  delete **

## 5. Rerun after delete：

1.  **Execute run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 -- masters 172.16.0.165,172.16.0.166,172.16.0.167 -- nodes 172.16.0.168,172.16.0.169 --passwd  (executed successfully）**

```shell
[root@baremetal002 ~]# kubectl get nodes -A
NAME           STATUS   ROLES    AGE     VERSION
baremetal001   Ready    <none>   46s     v1.19.9
baremetal002   Ready    master   2m21s   v1.19.9
baremetal003   Ready    master   98s     v1.19.9
baremetal004   Ready    <none>   45s     v1.19.9
```

2. **Execute sealer delete -f /root/.sealer/my-cluster/Clusterfile**



# B. Not executed in master0：   4C4G centos7.9

1.

| masters | 172.16.0.165,172.16.0.166,172.16.0.167 |
| ------- | -------------------------------------- |
| nodes   | 172.16.0.168,172.16.0.169              |

**Execute on node 172.16.0.164:**：

**sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters 172.16.0.165,172.16.0.166,172.16.0.167 --nodes 172.16.0.168,172.16.0.169 --passwd sealer123（executed successfully）**

```shell
[root@test-baremetal008 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE    VERSION
test-baremetal003   Ready    master   87s    v1.19.9
test-baremetal004   Ready    master   2m6s   v1.19.9
test-baremetal005   Ready    master   33s    v1.19.9
test-baremetal006   Ready    <none>   15s    v1.19.9
test-baremetal007   Ready    <none>   14s    v1.19.9
```

## 2. Scale up

**masters add 172.16.0.170，172.16.0.171**

**nodes add 172.16.0.172**

1. **Modify Clusterfile：**

```shell
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.165
    - 172.16.0.166
    - 172.16.0.167
    - 172.16.0.170//masters adds two node
    - 172.16.0.171
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.168
    - 172.16.0.169
    - 172.16.0.172//nodes adds one node
```

2. **execute: sealer apply -f /root/.sealer/my-cluster/Clusterfile （executed successfully）**

![image.png](https://intranetproxy.alipay.com/skylark/lark/0/2021/png/7656565/1622787574460-e4d3e59b-6025-400d-81a2-0ee08171864a.png)

```shell
[root@test-baremetal008 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE     VERSION
test-baremetal001   Ready    <none>   117s    v1.19.9
test-baremetal002   Ready    master   2m14s   v1.19.9
test-baremetal003   Ready    master   8m58s   v1.19.9
test-baremetal004   Ready    master   9m37s   v1.19.9
test-baremetal005   Ready    master   8m4s    v1.19.9
test-baremetal006   Ready    <none>   7m46s   v1.19.9
test-baremetal007   Ready    <none>   7m45s   v1.19.9
test-baremetal009   Ready    master   2m52s   v1.19.9
```

## 3. scale down

​     **Delete 172.16.0.170, 172.16.0.171 masters and 172.16.0.172 nodes**

1. **Modify Clusterfile**：

```shell
  masters:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.165
    - 172.16.0.166
    - 172.16.0.167  
    memory: "4"
    systemDisk: "100"
  network:
    cniName: calico
    interface: eth0
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
  nodes:
    count: "3"
    cpu: "4"
    dataDisks:
    - "100"
    ipList:
    - 172.16.0.168
    - 172.16.0.169 
```



2. **execute Sealer /root/.sealer/my-cluster/Clusterfile (Successfully executed）：**

```
[root@test-baremetal008 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE   VERSION
test-baremetal003   Ready    master   16m   v1.19.9
test-baremetal004   Ready    master   16m   v1.19.9
test-baremetal005   Ready    master   15m   v1.19.9
test-baremetal006   Ready    <none>   14m   v1.19.9
test-baremetal007   Ready    <none>   14m   v1.19.9
```

**Problem: (same as the problem in Master0 node execution scale down) :**

**The Scale Down node needs to remove rootfs as well as the Seautil tool**

```shell
[root@test-baremetal008 ~]# ssh root@172.16.0.170 //Enter the node that has been scaled down
Welcome to Alibaba Cloud Elastic Compute Service !
[root@test-baremetal009 ~]# ls /var/lib/sealer/data/my-cluster/rootfs/ 
bin/                      cri/                      images/                   Kubefile                  README.md                 scripts/                  TREE.md                   
cni/                      etc/                      kubeadm-join-config.yaml  Metadata                  registry/                 statics/                  
```

## 4. Delete

​	**execute ： sealer delete -f /root/.sealer/my-cluster/Clusterfile （Delete successfully）：**

## 5. Rerun after delete：

​	**execute： sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters    172.16.0.165,172.16.0.166,172.16.0.167 --nodes 172.16.0.168,172.16.0.169 --passwd Seadent23（executed successfully）**

```shell
[root@test-baremetal008 ~]# kubectl get nodes -A
NAME                STATUS   ROLES    AGE   VERSION
test-baremetal003   Ready    master   12m   v1.19.9
test-baremetal004   Ready    master   12m   v1.19.9
test-baremetal005   Ready    master   11m   v1.19.9
test-baremetal006   Ready    <none>   11m   v1.19.9
test-baremetal007   Ready    <none>   11m   v1.19.9
```

