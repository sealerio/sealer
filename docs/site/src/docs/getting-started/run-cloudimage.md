# Cluster Lifecycle Management

## Start the Kubernetes cluster

### Sealer run command line

One command deploy an entire kubernetes cluster.

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

More examples :

run cloud image on an exists servers

Server ip address| 192.168.0.1 ~ 192.168.0.13
---|---
**server password**  | **sealer123**

*Run the kubernetes cluster on the local server.*

```shell
sealer run kubernetes:v1.19.8 \
   -m 192.168.0.1,192.168.0.2,192.168.0.3 \
   -n 192.168.0.4,192.168.0.5,192.168.0.6 \
   -p sealer123 # ssh passwd
```

*Check the Cluster*

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

## Scale up and down the Kubernetes cluster

*Using join command to scale up the local server.*

```shell script
$ sealer join \
   --masters 192.168.0.7,192.168.0.8,192.168.0.9,192.168.0.10 \
   --nodes 192.168.0.11,192.168.0.12,192.168.0.13
# or
$ sealer join --masters 192.168.0.7-192.168.0.10 --nodes 192.168.0.11-192.168.0.13
```

*Using delete command to scale down the local server.*

```shell
$ sealer delete \
   --masters 192.168.0.7,192.168.0.8,192.168.0.9,192.168.0.10 \
   --nodes 192.168.0.11,192.168.0.12,192.168.0.13
# or
$ sealer delete --masters 192.168.0.7-192.168.0.10 --nodes 192.168.0.11-192.168.0.13
```

## Upgrade the Kubernetes cluster

Specify which image you want to use for upgrading as well as the cluster name you want to upgrade via a flag "-c".

```shell script
sealer upgrade registry.cn-beijing.aliyuncs.com/sealer-io/kubernetes:v1.19.9_develop -c my-cluster
```

if the flag "-c" is missed,sealer will use the default cluster name instead.

## Clean up the Kubernetes cluster

```shell
sealer delete --all
```

Sealer will also remove infrastructure resources if you use cloud mod.