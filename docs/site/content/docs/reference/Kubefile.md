+++
title = "Configure the Kubefile"
description = "Answers to frequently asked questions."
date = 2021-05-01T19:30:00+00:00
updated = 2021-05-01T19:30:00+00:00
draft = false
weight = 30
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Answers to frequently asked questions."
toc = true
top = false
+++

A `Kubefile` is a text document that contains all the commands a user could call on the command line to assemble an
image.We can use the `Kubefile` to define a cluster image that can be shared and deployed offline. a `Kubefile` just
like `Dockerfile` which contains the build instructions to define the specific cluster.

# Kubefile instruction

## FROM instruction

The `FROM` instruction defines which base image you want reference, and the first instruction in Kubefile must be the
FROM instruction. Registry authentication information is required if the base image is a private image. By the way
official base images are available from the Sealer community.

> command format：FROM {your base image name}

USAGE：

For example ,use the base image `kubernetes:v1.19.8` which provided by the Sealer community to build a new cloud image.

`FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`

## COPY instruction

The `COPY` instruction used to copy the contents from the context path such as file or directory to the `rootfs`. all
the cloud image is based on the [rootfs](../../../../api/cloudrootfs.md), and the default src path is
the `rootfs` .If the specified destination directory does not exist, sealer will create it automatically.

> command format：COPY {src dest}

USAGE：

For example , copy `mysql.yaml`to`rootfs/mysql.yaml`

`COPY mysql.yaml .`

For example , copy directory `apollo` to `rootfs/charts/apollo`

`COPY apollo charts`

## RUN instruction

The RUN instruction will execute any commands in a new layer on top of the current image and commit the results. The
resulting committed image will be used for the next step in the `Kubefile`.

> command format：RUN {command args ...}

USAGE：

For example ,Using `RUN` instruction to execute a commands that download kubernetes dashboard.

`RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml`

### CMD instruction

The format of CMD instruction is similar to RUN instruction, and also will execute any commands in a new layer. However,
the CMD command will be executed when the cluster is started . it is generally used to start applications or configure
the cluster. and it is different with `Dockerfile` CMD ,If you list more than one CMD in a `Kubefile` ,then all of them
will take effect.

> command format：CMD {command args ...}

USAGE：

For example ,Using `CMD` instruction to execute a commands that apply the kubernetes dashboard yaml.

`CMD kubectl apply -f recommended.yaml`
