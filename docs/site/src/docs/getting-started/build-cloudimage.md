# Build CloudImage

## Build command line

You can run the build command line after sealer installed. The current path is the context path ,default build type is
`lite` and use build cache.

```shell
sealer build [flags] PATH
```

Flags:

```shell
Flags:
      --base                build with base image,default value is true. (default true)
      --build-arg strings   set custom build args
  -h, --help                help for build
  -t, --imageName string    cluster image name
  -f, --kubefile string     kubefile filepath (default "Kubefile")
  -m, --mode string         cluster image build type, default is lite (default "lite")
      --no-cache            build without cache
```

## Build instruction

### FROM instruction

FROM: Refers to a base image, and the first instruction in the Kubefile must be a FROM instruction. If the base image is
a private repository image, the repository authentication information is required, and the sealer community also
provides an official base image for use.

> instruction format: FROM {your base image name}

Examples:

use `kubernetes:v1.19.8` which provided by sealer community as base image。

`FROM registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8`

### COPY instruction

COPY: Copy files or directories in the build context to rootfs。

The cluster image file structure is based on the rootfs structure. The default target path is rootfs, and it will be
automatically created when the specified target directory does not exist.

> instruction format: COPY {src dest}

Examples:

Copy mysql.yaml to the rootfs directory.

`COPY mysql.yaml .`

Copy the executable binary "helm" to the system $PATH.

`COPY helm ./bin`

Copy remote web file or git repository to cloud image.

`COPY https://github.com/alibaba/sealer/raw/main/applications/cassandra/cassandra-manifest.yaml manifests`

Support wildcard copy, copy all yaml files in the test directory to rootfs manifests directory.

`COPY test/*.yaml manifests`

### ARG instruction

ARG: Supports setting command line parameters in the build phase for use with CMD and RUN instruction。

> instruction format: ARG <parameter name>[=<default value>]

Examples:

```shell
FROM kubernetes:v1.19.8
# set default version is 4.0.0, this will be used to install mongo application.
ARG Version=4.0.0
# mongo dir contains many mongo version yaml file.
COPY mongo manifests
# arg Version can be used with RUN instruction.
RUN echo ${Version}
# use Version arg to install mongo application.
CMD kubectl apply -f mongo-${Version}.yaml
```

This means run `kubectl apply -f mongo-4.0.0.yaml` in the CMD instruction.

### RUN instruction

RUN: Use the system shell to execute the build command,accept multiple command parameters, and save the command
execution result during build. If the system command does not exist, this instruction will return error.

> instruction format: RUN {command args ...}

Examples:

Use the wget command to download a kubernetes dashboard。

`RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml`

### CMD instruction

CMD: Similar to the RUN instruction format, use the system shell to execute build commands. However, the CMD command
will be executed when sealer run, generally used to start and configure the cluster. In addition, unlike the CMD
instructions in the Dockerfile, there can be multiple CMD instructions in a kubefile.

> instruction format: CMD {command args ...}

Examples:

Install a kubernetes dashboard using the kubectl command。

`CMD kubectl apply -f recommended.yaml`

## Build type

Currently, sealer build supports three build approaches for different requirement scenarios.

### 1.cloud build mode

The default build type. Based on cloud (currently only supported by Ali Cloud, welcome to contribute other cloud
providers), sealer can automatically create infra resources, deploy Kubernetes cluster and build images. And cloud Build
is the most compatible construction method and can basically meet the construction requirements of 100%. This build
approach is recommended if you are delivering a cloud image that involves infra resources such as persistence storage.
But the downside is that there is a cost associated with creating the cloud resources.

For example ,log in to the image registry, and create the build context directory,then put the files required for
building the image . In Cloud build, sealer will pull up the cluster and send the context to the cloud to build an image
,also will push the image automatically.

```shell
[root@sea ~]# sealer login registry.cn-qingdao.aliyuncs.com -u username -p password
[root@sea ~]# mkdir build && cd build && mv /root/recommended.yaml .
[root@sea build]# vi Kubefile
[root@sea build]# cat Kubefile
FROM kubernetes:v1.19.8
COPY recommended.yaml .
CMD kubectl apply -f recommended.yaml
[root@sea build]# ls
Kubefile  recommended.yaml
#start to build
[root@sea build]# sealer build -t registry.cn-qingdao.aliyuncs.com/sealer-io/my-cluster:v1.19.9 .
```

### 2.container build mode

Similar to the cloud build mode, we can apply a Kubernetes cluster by starting multiple Docker containers as Kubernetes
nodes ( simulating cloud ECS), which consume very few resources to complete the build instruction. The disadvantage of
the container build is that some scenarios which rely on the infra resources is not supported very well.

You can specify the build type with the '-m container' argument to use container build.

```shell
sealer build -m container -t my-cluster:v1.19.9 .
```

### 3.lite build mode

The lightest build mode. By parsing the helm Chart, submitting the image list, parsing the kubernetes resource file
under the manifest to build the cloud image. and this can be done without starting the cluster

The advantages of this build mode is the lowest resource consumption . Any host installed sealer can use this mode to
build cloud image.

The disadvantage is that some scenarios cannot be covered. For example, the image deployed through the operator cannot
be obtained, and some images delivered through proprietary management tools are also can not be used.

In addition, some command such as `kubectl apply` or `helm install` will execute failed because the lite build process
will not pull up the cluster, but it will be saved as a layer of the image in the build stage.

Lite build is suitable for the scenarios where there is a list of known images or no special resource requirements.

Kubefile example:

```shell
FROM kubernetes:v1.19.8
COPY imageList manifests
COPY apollo charts
COPY helm /bin
CMD helm install charts/apollo
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

As in the above example, the lite build will parse and cache the list of images to the registry from the following three
locations:

* `manifests/imageList`: The content is a list of images line by line, If this file exists, will be extracted to the
  desired image list . The file name of `imageList` must be fixed, unchangeable, and must be placed under manifests.

* `manifests` directory: Lite build will parse all the yaml files in the manifests directory and extract it to the
  desired image list.

* `charts` directory: this directory contains the helm chart, and lite build will resolve the image address from the
  helm chart through the helm engine.

You can specify the build type with the '-m lite' argument to use lite build.

```shell
sealer build -m lite -t my-cluster:v1.19.9 .
```

## Build arg

If the user wants to customize some parameters in the build stage, or in the image startup stage. could
set `--build-arg` or write `ARG` in the Kubefile.

### used build arg in Kubefile

examples:

```shell
FROM kubernetes:v1.19.8
# set default version is 4.0.0, this will be used to install mongo application.
ARG Version=4.0.0
# mongo dir contains many mongo version yaml file.
COPY mongo manifests
# arg Version can be used with RUN instruction.
RUN echo ${Version}
# use Version arg to install mongo application.
CMD kubectl apply -f mongo-${Version}.yaml
```

this will use `ARG` value 4.0.0 to build the image.

```shell
sealer build -t my-mongo:v1 .
```

### use build arg in sealer build command line

examples:

use `--build-arg` value to overwrite the `ARG` value set in the kuebfile. this will install mongo application with
version 4.0.7.

```shell
sealer build -t my-mongo:v1 --build-arg Version=4.0.7 .
```

### use build arg in sealer run command line

examples:

use `--cmd-args` to overwrite the `ARG` value of CMD instruction set in the kuebfile. this will install mongo
application equals run  `kubectl apply -f mongo-5.1.1.yaml`.

```shell
sealer run --cmd-args Version=5.1.1 -m 172.16.0.227 -p passsword my-mongo:v1
```

### use build arg in Clusterfile

examples:

use `cmd_args` fields to overwrite the `ARG` value of CMD instruction set in the kuebfile. this will install mongo
application equals run  `kubectl apply -f mongo-4.9.0.yaml`.

```yaml
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  cmd_args:
    - Version=4.9.0
  hosts:
    - ips:
        - 172.16.0.227
  image: my-mongo:v1
  ssh:
    passwd: passsword
    pk: /root/.ssh/id_rsa
    port: "22"
    user: root
```

```shell
sealer apply -f Clusterfile
```

## More build examples

### lite build:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 .`

### container build:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 -m container .`

### cloud build:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 --mode cloud .`

### build without cache:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 --no-cache .`

### build without base:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 --base=false .`

### build with args:

`sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .`

### build with private image registry

#### different registry have different users

just to login,for example :

`sealer login registry.cn-qingdao.aliyuncs.com -u username -p password`

#### same registry have different users

you need to write the credential file named at "imageListWithAuth.yaml" in your build context. and its format like
below, it is still possible to trigger sealer build to pull docker images, works like `COPY imageList manifests`.

```yaml
- registry: registry.cn-shanghai.aliyuncs.com
  username: user1
  password: pw
  images:
    - registry.cn-shanghai.aliyuncs.com/xxx/xxx1:v1.1
    - registry.cn-shanghai.aliyuncs.com/xxx/xxx2:v1.1
- registry: registry.cn-shanghai.aliyuncs.com
  username: user2
  password: pw
  images:
    - registry.cn-shanghai.aliyuncs.com/xxx/xxx3:v1.1
    - registry.cn-shanghai.aliyuncs.com/xxx/xxx4:v1.1
```

filed "registry" is optional , if not present sealer will use the default "docker.io" as its domain name. below is
example build context: this will trigger pull images form `imageList` and `imageListWithAuth.yaml`.

For example:

```shell
[root@iZbp16ikro46xwgqzij67sZ build]# ll
total 12
-rw-r--r-- 1 root root   7 Feb 28 14:10 imageList
-rw-r--r-- 1 root root 450 Mar  1 10:20 imageListWithAuth.yaml
-rw-r--r-- 1 root root  49 Feb 28 14:06 Kubefile
[root@iZbp16ikro46xwgqzij67sZ build]#
[root@iZbp16ikro46xwgqzij67sZ build]# cat Kubefile
FROM kubernetes:v1.19.8
COPY imageList manifests
```

## Base image list

### base image with sealer docker

|image name |platform| kubernetes version|docker version|
--- | --- | ---| ---|
|kubernetes:v1.19.8|AMD| 1.19.8|19.03.14|
|kubernetes-arm64:v1.19.7|ARM| 1.19.7|19.03.14|

### base image with native docker

|image name |platform| kubernetes version|docker version|
--- | --- | ---| ---|
|kubernetes-kyverno:v1.19.8|AMD| 1.19.8|19.03.15|
|kubernetes-kyverno-arm64:v1.19.7|ARM| 1.19.7|19.03.15|