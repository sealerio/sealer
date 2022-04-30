# Use sealer in container

## Motivations

We can use docker containers running as IAAS infra or an isolated environment to test cloud image or build a new one
which consumes very few resources.

## Prerequisites

The machine needs to install docker in advance, and ensure that it can connect to the external network.

## Examples

### Use container as IAAS infra

We have built a container image based on
this [Dockerfile](https://github.com/sealerio/sealer/tree/main/pkg/infra/container/imagecontext/base/Dockerfile)

named `registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image:latest` to simulate the ECS virtual machine.

you can pull it via `docker pull registry.cn-qingdao.aliyuncs.com/sealer-io/sealer-base-image:latest`

#### run sealer build in container mode

You can specify the build type with the '-m container' argument to use container build.

```shell
sealer build -m container -t my-cluster:v1.19.9 .
```

#### apply a cluster in container

You can apply a Kubernetes cluster by starting multiple docker containers as Kubernetes nodes ( simulating cloud ECS),
set the number of masters or nodes, and set provider "CONTAINER":

```shell
sealer run kubernetes:v1.19.8 --masters 3 --nodes 3 --provider CONTAINER
```

### Use container as build machine

Use this [Dockerfile](https://github.com/sealerio/sealer/tree/main/pkg/infra/container/imagecontext/build/Dockerfile)
to build your own docker image.

Example:

```shell
docker build -t base:v1 -f Dockerfile .
```

start the container:

```shell
docker run --rm --name master1 --privileged --volume /lib/modules:/lib/modules:ro --volume /var/lib/sealer:/var/lib/sealer --detach --tty base:v1
```

In container "master1" you can run sealer build:

1. copy sealer binary to container master1 and enter the container

```shell
docker cp /usr/local/bin/sealer master1:/usr/local/bin/sealer
docker exec -it master1 /bin/bash
```

2. write the Kubefile in container "master1" and build the cloud image

```shell
sealer build -f Kubefile -t my-images:v1 .
```