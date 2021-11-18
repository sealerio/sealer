# sealer plugin

Plugins can help users do some peripheral things, like change hostname, upgrade kernel, or add node label...

## hostname plugin

If you write the plugin config in Clusterfile and apply it, sealer will help you to change all the hostnames

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: hostname
spec:
  type: HOSTNAME
  data: |
     192.168.0.2 master-0
     192.168.0.3 master-1
     192.168.0.4 master-2
     192.168.0.5 node-0
     192.168.0.6 node-1
     192.168.0.7 node-2
```

## shell plugin

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: shell
spec:
  type: SHELL
  action: PostInstall
  on: 192.168.0.2-192.168.0.4 #or 192.168.0.2,192.168.0.3,192.168.0.7
  data: |
     kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule
```

## label plugin

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: label
spec:
  type: LABEL
  data: |
     192.168.0.2 ssd=true
     192.168.0.3 ssd=true
     192.168.0.4 ssd=true
     192.168.0.5 ssd=false,hdd=true
     192.168.0.6 ssd=false,hdd=true
     192.168.0.7 ssd=false,hdd=true
```

## clusterCheck plugin

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: checkCluster
spec:
  type: CLUSTERCHECK
  action: PreGuest
```  
