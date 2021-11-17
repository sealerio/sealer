# Raw docker CloudImage

## Motivations

The existing base images mostly use customized docker,but many k8s clusters use raw docker as container runtime.So it's necessary to provide a base image with raw docker,this page is a guide of how to get a base image with raw docker.

## Use cases

### Step 1

get a image which you will modify it later, you may think it as your base image.To demonstrate the workflow,I will use `kubernetes:v1.19.8`.you can get the same image by executing `sealer pull kubernetes:v1.19.8`.

### Step 2

find the image layer id by executing `sealer inspect kubernetes:v1.19.8`.there are four layers in this image,and you will only use two of them.the first one's id is `c1aa4aff818df1dd51bd4d28decda5fe695bea8a9ae6f63a8dd5541c7640b3d6`,it consist of bin files,config files,registry files,scripts and so on.(I will use {layer-id-1} to refer to it in the following，Actually，it's a sha256 string),and the another one's id is `991491d3025bd1086754230eee1a04b328b3d737424f1e12f708d651e6d66860`,it consist of network component yaml files.(I will use {layer-id-2} to refer to it in the following，Actually，it's also a sha256 string)

### Step 3

choose a raw docker binary version from `https://download.docker.com/linux/static/stable/x86_64/` if your marchine is based on x86_64 architecture,and download it.(other architecture can be find at `https://download.docker.com/linux/static/stable/`)

### Step 4

replace `/var/lib/sealer/data/overlay2/{layer-id-1}/cri/docker.tar.gz` with the file you download in step 3. **Attention** that you should make sure the file name is  same as 'docker.tar.gz' after replacement.

### Step 5

pull the official "registry" image and replace existing customized "registry" image at `/var/lib/sealer/data/overlay2/{layer-id-1}/etc/registry.tar`.Firstly make sure raw docker have already installed,then execute `docker pull registry:2.7.1 && docker save -o registry.tar registry:2.7.1 && mv registry.tar /var/lib/sealer/data/overlay2/{layer-id-1}/etc/registry.tar`

### Step 6

edit the file 'daemon.json' at `/var/lib/sealer/data/overlay2/{layer-id-1}/etc/`,delete the `mirror-registries` attribute。

### Step 7

Now the base image still need network components to create k8s clusters,so move the file "tigera-operator.yaml" and the file "custom-resources.yaml" from `/var/lib/sealer/data/overlay2/{layer-id-2}/etc/` to `/var/lib/sealer/data/overlay2/{layer-id-1}/etc/`

### Step 8

switch to directory `/var/lib/sealer/data/overlay2/{layer-id-1}/` and build image   by execute `sealer build --mode lite -t <image-name:image-tag> .` , edit the `Kubefile` and make sure it's content is :

```shell script
FROM scratch
COPY . .
CMD kubectl apply -f etc/tigera-operator.yaml && kubectl apply -f etc/custom-resources.yaml
```

