# 集群生命周期管理

## 启动 Kubernetes 集群

### Sealer运行命令行

一个命令部署整个 Kubernetes 集群。

```shell
sealer run kubernetes:v1.19.8 -m 192.168.0.1 -p password
```

Flags:

```shell
Flags:
      --cmd-args strings     set args for image cmd instruction
  -e, --env strings          set custom environment variables
  -h, --help                 help for run
  -m, --masters string       set Count or IPList to masters
  -n, --nodes string         set Count or IPList to nodes
  -p, --passwd string        set cloud provider or baremetal server password
      --pk string            set baremetal server private key (default "/root/.ssh/id_rsa")
      --pk-passwd string     set baremetal server private key password
      --port string          set the sshd service port number for the server (default port: 22) (default "22")
      --provider ALI_CLOUD   set infra provider, example ALI_CLOUD, the local server need ignore this
  -u, --user string          set baremetal server username (default "root")
```

更多示例：

在现有服务器上运行cloud image

Server ip address| 192.168.0.1 ~ 192.168.0.13
---|---
**server password**  | **sealer123**

*在本地服务器上运行 kubernetes 集群。*

```shell
sealer run kubernetes:v1.19.8 \
   -m 192.168.0.1,192.168.0.2,192.168.0.3 \
   -n 192.168.0.4,192.168.0.5,192.168.0.6 \
   -p sealer123 # ssh passwd
```

*检查集群*

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

## 扩展和缩减 Kubernetes 集群

*使用 join 命令扩展本地服务器。*

```shell script
$ sealer join \
   --masters 192.168.0.7,192.168.0.8,192.168.0.9,192.168.0.10 \
   --nodes 192.168.0.11,192.168.0.12,192.168.0.13
# or
$ sealer join --masters 192.168.0.7-192.168.0.10 --nodes 192.168.0.11-192.168.0.13
```

*使用 delete 命令缩减本地服务器。*

```shell
$ sealer delete \
   --masters 192.168.0.7,192.168.0.8,192.168.0.9,192.168.0.10 \
   --nodes 192.168.0.11,192.168.0.12,192.168.0.13
# or
$ sealer delete --masters 192.168.0.7-192.168.0.10 --nodes 192.168.0.11-192.168.0.13
```

## 升级 Kubernetes 集群

通过标志“-c”指定要用于升级的映像以及要升级的集群名称。

```shell script
sealer upgrade registry.cn-beijing.aliyuncs.com/sealer-io/kubernetes:v1.19.9_develop -c my-cluster
```

如果缺少标志“-c”，sealer 将使用默认集群名称。

## 清理 Kubernetes 集群

```shell
sealer delete --all
```

如果您使用cloud mod，Sealer还将删除基础设施资源。