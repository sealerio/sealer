# 快速开始

## 使用sealer创建一个kubernetes集群

```shell script
# 下载和安装sealer二进制
wget https://github.com/alibaba/sealer/releases/download/v0.7.1/sealer-v0.7.1-linux-amd64.tar.gz && \
tar zxvf sealer-v0.7.1-linux-amd64.tar.gz && mv sealer /usr/bin
# 运行一个六节点的kubernetes集群
sealer run kubernetes:v1.19.8 \
  --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
  --node 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx
```

```shell script
[root@iZm5e42unzb79kod55hehvZ ~]# kubectl get node
NAME                    STATUS ROLES AGE VERSION
izm5e42unzb79kod55hehvz Ready master 18h v1.19.8
izm5ehdjw3kru84f0kq7r7z Ready master 18h v1.19.8
izm5ehdjw3kru84f0kq7r8z Ready master 18h v1.19.8
izm5ehdjw3kru84f0kq7r9z Ready <none> 18h v1.19.8
izm5ehdjw3kru84f0kq7raz Ready <none> 18h v1.19.8
izm5ehdjw3kru84f0kq7rbz Ready <none> 18h v1.19.8
```

## 增加删除节点

```shell script
sealer join --masters 192.168.0.2,192.168.0.3,192.168.0.4
sealer join --nodes 192.168.0.5,192.168.0.6,192.168.0.7
```

## 清理集群

创建集群会默认创建一个Clusterfile存储在 /root/.sealer/[cluster-name]/Clusterfile, 里面包含集群元数据信息.

删除集群:

```shell script
sealer delete -f /root/.sealer/my-cluster/Clusterfile
# 或者
sealer delete --all
```

## 自定义集群镜像

上面我们看到的`kubernetes:v1.19.8`就是一个标准的集群镜像，有时我们希望在集群镜像中带一些我们自己自定义的组件，就可以使用此功能。

比如这里我们创建一个包含dashboard的集群镜像：

Kubefile:

```shell script
# 基础镜像中包含安装kuberntes的所有依赖，sealer已经制作好，用户直接使用它即可
FROM kubernetes:v1.19.8
# 下载dashboard的yaml文件
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
# 集群启动时的命令
CMD kubectl apply -f recommended.yaml
```

构建集群镜像:

```shell script
sealer build -t dashboard:latest .
```

运行集群镜像，这时运行出来的就是一个包含了dashboard的集群:

```shell script
# sealer会启动一个kubernetes集群并在集群中启动dashboard
sealer run dashboard:latest --masters 192.168.0.2 --passwd xxx
# 查看dashboard的pod
kubectl get pod -A|grep dashboard
```

## 把集群镜像推送到镜像仓库

```shell script
sealer tag dashboard:latest registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
sealer push registry.cn-qingdao.aliyuncs.com/sealer-io/dashboard:latest
```

## 镜像导入导出

```shell script
sealer save -o dashboard.tar dashboard:latest
# 可以把tar拷贝到客户环境中进行load
sealer load -i dashboard.tar
```