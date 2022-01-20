# Run a cluster

## Run on exist servers

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

### scale up and down

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

## Clean up the Kubernetes cluster

```shell
sealer delete --all
```

Sealer will also remove infrastructure resources if you use cloud mod.