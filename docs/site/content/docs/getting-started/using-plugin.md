+++
title = "Plugin"
description = "Using plugin to do some edge works"
date = 2021-05-01T08:20:00+00:00
updated = 2021-05-01T08:20:00+00:00
draft = false
weight = 23
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Plugins can help users do some peripheral things, like change hostname, upgrade kernel, or add node label..."
toc = true
top = false
+++

# Plugins Usage

Set Plugins metadata in Clusterfile and apply it~

For example, set node label after install kubernetes cluster:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: kubernetes:v1.19.9
  provider: BAREMETAL
  ssh:
    passwd:
    pk: xxx
    pkPasswd: xxx
    user: root
  network:
    podCIDR: 100.64.0.0/10
    svcCIDR: 10.96.0.0/22
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
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: LABEL
spec:
  data: |
     172.20.126.8 ssd=false,hdd=true
```

```shell script
sealer apply -f Clusterfile
```

## hostname plugin

HOSTNAME plugin will help you to change all the hostnames

```yaml
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: HOSTNAME # should not change this name
spec:
  data: |
     192.168.0.2 master-0
     192.168.0.3 master-1
     192.168.0.4 master-2
     192.168.0.5 node-0
     192.168.0.6 node-1
     192.168.0.7 node-2
```

## shell plugin

You can exec any shell command on specify node in any phase.

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall # PreInit PreInstall PostInstall
  on: 192.168.0.2-192.168.0.4 #or 192.168.0.2,192.168.0.3,192.168.0.7
  data: |
     kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule
```

action: the phase of command.

* PreInit: before init master0.
* PreInstall: before join master and nodes.
* PostInstall: after join all nodes.

on: exec on witch node.

## label plugin

Help you set label after install kubernetes cluster.

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: LABEL
spec:
  data: |
     192.168.0.2 ssd=true
     192.168.0.3 ssd=true
     192.168.0.4 ssd=true
     192.168.0.5 ssd=false,hdd=true
     192.168.0.6 ssd=false,hdd=true
     192.168.0.7 ssd=false,hdd=true
```

## Etcd backup

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: ETCD_BACKUP
spec:
  action: Manual
```

Etcd backup plugin is triggered manually: `sealer plugin -f etcd_backup.yaml`

## develop you own plugin

The plugin [interface](https://github.com/alibaba/sealer/blob/main/plugin/plugin.go)

```golang
type Interface interface {
	Run(context Context, phase Phase) error
}
```

[Example](https://github.com/alibaba/sealer/blob/main/plugin/labels.go):

```golang
func (l LabelsNodes) Run(context Context, phase Phase) error {
	if phase != PhasePostInstall {
		logger.Debug("label nodes is PostInstall!")
		return nil
	}
	l.data = l.formatData(context.Plugin.Spec.Data)

	return err
}
```

Then regist you [plugin](https://github.com/alibaba/sealer/blob/main/plugin/plugins.go):

```golang
func (c *PluginsProcesser) Run(cluster *v1.Cluster, phase Phase) error {
	for _, config := range c.Plugins {
		switch config.Name {
		case "LABEL":
			l := LabelsNodes{}
			err := l.Run(Context{Cluster: cluster, Plugin: &config}, phase)
			if err != nil {
				return err
			}
        // add you plugin here
		default:
			return fmt.Errorf("not find plugin %s", config.Name)
		}
	}
	return nil
}
```