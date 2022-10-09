# What is a Kubefile

## Introduction

Kubefile is a declaration for a set of kube-based applications and operations right after initiation of cluster.
Content of Kubefile is composed of various instructions and arguments. These declarations define what kind of
cluster-runtime and which applications will be packing together to distribute a user-specific cluster. Kubefile
serves for building ClusterImages, and building is available on `linux platform only`. The reason for naming it
as Kubefile is that the Kubefile contains a prefix 'Kube', which means that there are always Kubernetes-related
binaries located in every ClusterImage. And Kubefile aims to provide editor a sense of defining applications
based on taking k8s as kernel.

## Benefit

Kubefile enables greater flexibility and portability of distributed applications.
IT orgainizations take advantages fo Kubefiles to package distributed applications
and their dependencies in a Kubernetes cluster that can run on premise, in public
or private cloud, or on bare metal. Besides, Kubefile brings much more efficiency
in distributed application delivery lifecycle.

Without Kubefile, IT engineers take quite complicated steps to run distributed
application(s). Taking a three-node etcd cluster without Helm as an example, IT engineers must
prepare three nodes which meet the reprequisites of etcd installation, download
and configure essential dependencies of etcd, and start to install and construct
an etcd cluster. If etcd cluster is to be installed by Helm on an existing Kubernets,
it could be shorten steps above without concerning about reprequisites and manually
startup trigger. If there is no existing Kubernetes, Kubefile and ClusterImage are
the best to choice to setup etcd cluster with one command just in minutes.

## Concept Origin

Kubefile has quite similar concept with Dockerfile. And Kubefile is indeed inpired
by Dockerfile. Actullay, Dockerfile encapsulates single-host application(s) into a single-host
box(Docker image). While Kubefile encapsulates cluster-level distributed
application(s) into a cluster-level box(ClusterImage).

Docker uses Dockerfile to build Docker image, and run Docker container with Docker image.

sealer uses Kubefile to build ClusterImage, and run Kubernetes cluster with ClusterImage.

Docker tackles almost all of delivery issues of single-host application(s). sealer expands
the concept to cluster level, and mostly focus on the perspetives of distributed application(s).
To be specific, since Kubefile contains a Kubernetes cluster inside of itself by deault and
Kubernetes has powerful ability to manage containers from Dockerfiles, we can say that
Kubefile's concept in built on the basis of Dockerfile, Kubernetes.

## Kubefile Command Syntax

The recommend commands in Kubefile are as following:

```
FROM [scratch/kubernetes:v1.19.8] (specify a base image)
CNI  (specify a CNI to be deployed in cluster)
CSI  (specify a CSI to be deployed in cluster)
APP  (declare apps to install)
LAUNCH [applist]
```

`CNI`, `LAUNCH` are allowed to be declared once.

`APP`, `CSI` are allowed to be declared many times.

The further introductions for command as follows:

### FROM
`FROM` allows user to define the base image. User is able to do some incremental operations on
base image.

base image could be any of cluster image or scratch.

### CNI(Unrealized)

`CNI` allows user to define which CNI to be installed over Kubernetes.

Some behaviors are available:

`CNI --type calico / flannel --cidr ...` an easiest way to deploy CNI, sealer will support several types of CNI.

`CNI path / https://... / oss://`. `path` is a relative path refers to local build context, this can be a directory
contains multiple yaml or a single yaml for deploying CNI. `https://...; oss://` are remote addresses for downloading related files to deploy CNI.

### CSI(Unrealized)

`CSI` allows user to define which CSI to be installed over Kubernetes.

Some behaviors are available:

`CSI --type alibaba-cloud-csi-driver` an easiest way to deploy CSI, sealer will support several types of CSI.

`CSI [path, https://, oss://]`. `path` is a relative path refers to local build context, this can be a directory
contains multiple yaml or a single yaml for deploying CSI. `https://...; oss://` are remote addresses for downloading related files to deploy CSI.

`[]` means user could declare several sources

### APP

`APP` allows user to specify which applications to be installed over Kubernetes.

The `APP` instruction has one form:

`APP APP_NAME scheme:path1 scheme:path2`.

The `APP_NAME` is a unique name to kube image.

The `scheme` has three forms:

* `local://path_rel_2_build_context` (files are from build context, path is relative to build context)
* `http://example.yaml`
* `https://example.yaml`
* `helm://` (unrealized)

There can be many `APP` in a `Kubefile`.

### LAUNCH

`LAUNCH` allows user to specify which apps(specified by instruction `APP`) to start right after the completion of cluster initiation.
Users are able to declare which applications to launch within the sealer image.

The `LAUNCH` instruction has one form:

`LAUNCH APP_1 APP_2`

`LAUNCH ["APP_1", "APP_2"]`

There are two behaviors for `LAUNCH`:

* `helm install APP_1 APP_1_PATH_REL_2_Rootfs` (APP_1 is detected as helm app)
* `kubectl apply -f APP_2_PATH_REL_2_Rootfs` (APP_2 is detected as raw yaml set app)

The behaviors will be generated automatically, users don't have to care about that.

There can be only one `LAUNCH` or `CMDS` instruction in a `Kubefile`.

### CMDS

`CMDS` allows user to specify the self-defined executing commands at a cluster startup.

The `CMDS` instruction has one form:

`CMDS ["cmd1", "cmd2"]`

symbol `""` is necessary for `CMDS` instruction.

There can be only one `LAUNCH` or `CMDS` instruction in a `Kubefile`.
