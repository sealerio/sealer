# Quick start

## Install a kubernetes cluster

```shell script
# install Sealer binaries
wget https://github.com/alibaba/sealer/releases/download/v0.5.2/sealer-v0.5.2-linux-amd64.tar.gz && \
tar zxvf sealer-v0.5.2-linux-amd64.tar.gz && mv sealer /usr/bin
# run a kubernetes cluster
sealer run kubernetes:v1.19.9 --masters 192.168.0.2 --passwd xxx
```

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

## clean the cluster

Some information of the basic settings will be written to the Clusterfile and stored in /root/.sealer/[cluster-name]/Clusterfile.

```shell script
sealer delete -f /root/.sealer/my-cluster/Clusterfile
```

## Build your own CloudImage

For example, build a dashboard CloudImage:

Kubefile:

```shell script
# base CloudImage contains all the files that run a kubernetes cluster needed.
#    1. kubernetes components like kubectl kubeadm kubelet and apiserver images ...
#    2. docker engine, and a private registry
#    3. config files, yaml, static files, scripts ...
FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9
# download kubernetes dashboard yaml file
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# when run this CloudImage, will apply a dashboard manifests
CMD kubectl apply -f recommended.yaml
```

Build dashobard CloudImage:

```shell script
sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest .
```

Run your kubernetes cluster with dashboard:

```shell script
# sealer will install a kubernetes on host 192.168.0.2 then apply the dashboard manifests
sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest --masters 192.168.0.2 --passwd xxx
# check the pod
kubectl get pod -A|grep dashboard
```

## Push the CloudImage to the registry

```shell script
# you can push the CloudImage to docker hub, Ali ACR, or Harbor
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```
