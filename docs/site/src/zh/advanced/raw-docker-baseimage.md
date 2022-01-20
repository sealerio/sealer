# Raw docker BaseImage

## Motivations

The existing base images mostly use customized docker, but many k8s clusters use raw docker as container runtime. So it's necessary to provide a base image with raw docker, this page is a guide of how to get a base image with raw docker.

## Use cases

### How to use it

We provide an official BaseImage which uses official raw docker as container runtime: `kubernetes-rawdocker:v1.19.8`. If you want to create a k8s cluster, you can use it directly as `sealer run` command's argument or write it into your Clusterfile. If you want to use it as the base image to build other images by `sealer build`, `FROM kubernetes-rawdocker:v1.19.8` should be the first line in your Kubefile.

### How to build raw docker BaseImage

#### Step 1：choose a base image

Get an image which you will modify it later, you may think it as your base image. To demonstrate the workflow, I will use `kubernetes:v1.19.8`. You can get the same image by executing `sealer pull kubernetes:v1.19.8`.

#### Step 2: find the layers you will use later

Find the image layer id by executing `sealer inspect kubernetes:v1.19.8`. There are four layers in this image, and you will only use two of them. The first one's id is `c1aa4aff818df1dd51bd4d28decda5fe695bea8a9ae6f63a8dd5541c7640b3d6`, it consist of bin files, config files, registry files, scripts and so on. (I will use {layer-id-1} to refer to it in the following. Actually, it's a sha256 string) The another one's id is `991491d3025bd1086754230eee1a04b328b3d737424f1e12f708d651e6d66860`, it consist of network component yaml files. (I will use {layer-id-2} to refer to it in the following. Actually, it's also a sha256 string)

#### Step 3: get official raw docker

Choose a raw docker binary version from `https://download.docker.com/linux/static/stable/x86_64/` if your machine is based on x86_64 architecture, and download it. (other architecture can be found at `https://download.docker.com/linux/static/stable/`)

#### Step 4: replace sealer hacked docker

Replace `/var/lib/sealer/data/overlay2/{layer-id-1}/cri/docker.tar.gz` with the file you download in step 3, Before replacement you should do some handles.  **Attention** that you should make sure after replacement the compressed file name and untarred working directory tree is same as before. In this case, you should untar the file you download in step 3, enter the `docker` directory and tar all files in this directory with an output file whose name is `docker.tar.gz`.

#### Step 5: replace sealer hacked registry

Pull the official "registry" image and replace existing customized "registry" image at `/var/lib/sealer/data/overlay2/{layer-id-1}/images/registry.tar`. Firstly make sure raw docker have already installed, then execute `docker pull registry:2.7.1 && docker save -o registry.tar registry:2.7.1 && mv registry.tar /var/lib/sealer/data/overlay2/{layer-id-1}/images/registry.tar`.

#### Step 6: modify daemon.json

Edit the file 'daemon.json' at `/var/lib/sealer/data/overlay2/{layer-id-1}/etc/`, delete the `mirror-registries` attribute.

#### Step 7: build rawdocker alpine image

Switch to directory `/var/lib/sealer/data/overlay2/{layer-id-1}/`, edit the `Kubefile` and make sure it's content is:

```shell script
FROM scratch
COPY . .
```

Then build image by execute `sealer build --mode lite -t kubernetes-rawdocker:v1.19.8-alpine .`.

#### Extension

#### Step 8: add network components to alpine image

Now the base image still need network components to make k8s clusters work well, here we provide a guide for adding calico as network components.
First of all, create a `rawdockerBuild` directory as your build environment. Then you should move the file "tigera-operator.yaml" and the file "custom-resources.yaml" from `/var/lib/sealer/data/overlay2/{layer-id-2}/etc/` to `rawdockerBuild/etc`. After that you still need modify some contents in those two files to make sure the pods they create will pull docker images from your private registry, which will make your k8s clusters still work well in offline situations. In this case, firstly add a map-key value in "custom-resources.yaml", the key is `spec.registry` and the value is `sea.hub:5000`, secondly modify all docker image names in "tigera-operator.yaml" from `<registry>/<repository>/<imageName>:<imageTag>` to `sea.hub:5000/<repository>/<imageName>:<imageTag>`.
Next create a `imageList` file at `rawdockerBuild` directory, with the following content:

- calico/cni:v3.19.1
- calico/kube-controllers:v3.19.1
- calico/node:v3.19.1
- calico/pod2daemon-flexvol:v3.19.1
- calico/typha:v3.19.1
- tigrea/operator:v1.17.4

They are all the images needed to create network components, make sure that the tag is consistent with declared in the yaml file "tigera-operator.yaml" and "custom-resources.yaml".

#### Step 9: build rawdocker image

Switch to directory `rawdockerBuild`, create a `Kubefile` and make sure it's content is:

```shell script
FROM kubernetes-rawdocker:v1.19.8-alpine
COPY imageList manifests
COPY etc .
CMD kubectl apply -f etc/tigera-operator.yaml && kubectl apply -f etc/custom-resources.yaml
```

Then build image by execute `sealer build --mode lite -t kubernetes-rawdocker:v1.19.8 .`.