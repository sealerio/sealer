+++
title = "Architecture"
description = "Show you the Architecture and core modules of sealer"
date = 2021-05-01T19:30:00+00:00
updated = 2021-05-01T19:30:00+00:00
draft = false
weight = 30
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Architecture"
toc = true
top = false
+++

# Architecture

![](https://user-images.githubusercontent.com/8912557/133879086-f13e3e37-65c3-43e2-977c-e8ebf8c8fb34.png)

Sealer has two top module: Build Engine & Apply Engine

The Build Engine Using Kubefile and build context as Input, and build a CloudImage that contains all the dependencies.
The Apply Engine Using Clusterfile to Init a cluster witch contains kubernetes and other applications.

## Build Engine

* Parser : parse Kubefile into image metadata
* Registry : push or pull the CloudImage
* Store : save CloudImage to local disks

### Builders

* Lite Builder, sealer will check all the manifest or helm chart, decode docker images in those files, and cache then into CloudImage
* Cloud Builder, sealer will create a Cluster using public cloud, and exec `RUN & CMD` command witch defined in Kubefile, then cache all the docker image in the Cluster.
* Container Builder, Using Docker container as a node, run kubernetes cluster in container then cache all the docker images

## Apply Engine

* Infra : manage infrastructure, like create VMs in public cloud then apply the cluster on top of it. Or using docker emulation nodes.
* Runtime : cluster installer implementation, like using kubeadm to installl cluster.
* Config : application config, like mysql username passwd or other configs, you can using Config overwrite any file you want.
* Plugin : plugin help us do some extra work, like exec a shell command before install, or add a label to a node after install.
* Debug : help us check the cluster is helth or not, found reason when things unexpected.

## Other modules

* Filesystem : Copy CloudRootfs files to all nodes
* Mount : mount CloudImage all layers together
* Checker : do some precheck and post check
* Command : a command proxy to do some task witch os don't have the command. Like ipvs or cert manager.
* Guest : manage user application layer, like exec CMD command defined in Kubefile.
