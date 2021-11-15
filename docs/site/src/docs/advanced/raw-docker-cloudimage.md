# Raw docker CloudImage

## Motivations

The existing base images mostly use customized docker,but many k8s clusters use raw docker as container runtime.So it's necessary to provide a base image with raw docker,this page is a guide of how to get a base image with raw docker.

## Use cases

### Step 1

get a image which you will modify it later, you may think it as your base image.To demonstrate the workflow,I will use `kubernetes:v1.19.9-alpine`.you can get the same image by executing `sealer pull kubernetes:v1.19.9-alpine`.

### Step 2

find the image layer id by executing `sealer inspect kubernetes:v1.19.9-alpine`.（I will use {layer-id} to refer to it in the following，Actually，it's a sha256 string）

### Step 3

choose a raw docker binary version from `https://download.docker.com/linux/static/stable/x86_64/` if your marchine is based on x86_64 architecture,and download it.(other architecture can be find at `https://download.docker.com/linux/static/stable/`)

### Step 4

replace `/var/lib/sealer/data/overlay2/{layer-id}/cri/docker.tar.gz` with the file you download in step 3. **Attention** that you should make sure the file name is  same as 'docker.tar.gz' after replacement.

### Step 5

edit the file 'daemon.json' at `/var/lib/sealer/data/overlay2/{layer-id}/etc/`,delete the `mirror-registries` attribute。

### Step 6

switch to directory `/var/lib/sealer/data/overlay2/{layer-id}/` and build image   by execute `sealer build --mode lite -t <image-name:image-tag> .` , there is already a `Kubefile` at this directory,so we don't need create a new one. The `Kubefile` content is :

```shell script
FROM scratch
COPY . .
```

