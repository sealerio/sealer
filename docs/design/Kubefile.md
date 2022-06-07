# What is a Kubefile

## Introduction

Kubefile is a script, composed of various commands(instructions) and arguments
listed successively to automatically perform an action on a base ClusterImage in
order to create(or form) a new ClusterImage. It is used by the sealer tool on
only Linux platform for building ClusterImage. The reason of naming it as Kubefile,
which contains the same prefix 'Kube' of Kubernetes, is that there is always
a Kubernetes binary located in each ClusterImage. And when ClusterImage is ran from
Kubefile, the built-in Kubernetes cluster will be setup and manage all infrastructure,
and provide orchestration ability to upper applications.

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

## Kubefile Example

Kubefile is always located somewhere on a machine, or in a git repository. It can be placed with binaries, packages,
YAML files and any others. And it can be placed alone as well. Since Kubefile is used to
build a ClusterImage and ClusterImage represents distributed application(s), there must be
something else which should be included except Kubefile. It is **building context**, the context of ClusterImage builing procedure.

Here is a Kubefile folder example:

![image](https://user-images.githubusercontent.com/9465626/168005080-a9b47180-6284-484c-93bc-717e4e5f490f.png)

There is a Kubefile and a `README.md` and `mysql-manifest.yaml` located in the
folder. When a ClusterImage is being built, the folder context is called
**building context**. In details, Kubefile has the following content:

```
FROM kubernetes:v1.18.3
COPY mysql-manifest.yaml .
CMD kubectl apply -f mysql-manifest.yaml
```

Like what we defined Kubefile at the top of this documentation, this Kubefile
contains three commands. Every command follows Kubefile syntax. Here are
brief illustration of these commands. `FROM kubernetes:v1.18.3` tells user that
the building ClusterImage is from a `kubernetes:v1.18.3` ClusterImage, or based on
`kubernetes:v1.18.3`. `COPY mysql-manifest.yaml .` means the building procedure
should `COPY` a YAML file from the building context into the newly building
ClusterImage. `CMD kubectl apply -f mysql-manifest.yaml` sets an executable
command of the ClusterImage, and when the ClusterImage runs up, the command will
be executed automatically. For more commands type illustration, please refer
to Kubefile Syntax.

## How to use Kubefile

It is quite easy for engineers to use Kubefile. Just one command of `sealer build -f KUBEFILE_PATH .`.
sealer is the tool which applies Kubefile and builds a ClusterImage from Kubefile.
In details, when engineer executes `sealer build` command, sealer binary will
start to run as a process, pass on demand building context into process context,
create a layer of OCI-compatible image for each command, and finally merge all
layers to be a complete ClusterImage. The ClusterImage could be stored locally and
pushed to remote image registries as well. In a word, engineer can use Kubefile
to build ClusterImage. Only ClusterImage can be run to organize a cluster directly.

## Kubefile Command Syntax

In the Kubefile example above, there are three kinds of Kubefile command type:
`FROM`, `COPY`, `CMD`. Actually there are much more command types in Kubefile
syntax. For more details about Kubefile command syntax, please refer to [Kubefile Command Syntax].