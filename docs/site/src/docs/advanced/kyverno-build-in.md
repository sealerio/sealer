# Kyverno build-in

## Motivations

It's common that some k8s clusters have their own private image registry, and they don't want to pull images from other registry for safety reasons. This page is about how to introduce kyverno in k8s cluster, which will redirect image pull request to Specified registry.

## Uses case

### How to use it

We provide an out-of-the-box cloud image which introduces kyverno into cluster:`kubernetes-raw_docerk-kyverno:v1.19.8`. Note that it contains no docker images other than those necessary to run a k8s cluster, so if you want to use this cloud image and you also need other docker images(such as `nginx`) to run a container, you need to cache the docker images to your private registry.

Of course `sealer` can help you do this,use `nginx` as an example.
Firstly include nginx in the file `imageList`.
You can execute `cat imageList` to make sure you have done this, and the result may seem like this:

```
 [root@ubuntu ~]# cat imageList
 nginx:latest
```

Secondly edit a Kubefile with the following content:

```
FROM kubernetes-raw_docerk-kyverno:v1.19.8
COPY imageList manifests
CMD kubectl run nginx --image=nginx:latest
```

Thirdly execute `sealer build` to build a new cloud image

```
 [root@ubuntu ~]# sealer build -t my-nginx-kubernetes:v1.19.8 .
```

Just a simple command and let sealer help you cache `nginx:latest` image to private registry. You may doubt whether sealer has successfully cached the image, please execute `sealer inspect my-nginx-kubernetes:v1.19.8` and locate the `layer` attribute of the `spec` section, you will find there are many layers. In this case, the last layer has two `key:value` pairs: `type: BASE`, `value: registry cache`, from which we know it's about images cached to registry. Remembering this layer's id, execute `cd /var/lib/sealer/data/overlay2/{layer-id}/registry/docker/registry/v2/repositories/library`, then you will find the nginx image existing in the directory.

Now you can use this new cloud image to create k8s cluster. After your cluster startup, there is already a pod running `nginx:latest` image, you can see it by execute `kubectl describe pod nginx`, and you can also create more pods running `nginx:latest` image.

### How to build kyverno BaseImage

The following is a sequence steps of building kyverno build-in cloud image

#### Step 1: choose a base image

Choose a base image which can create a k8s cluster with at least one master node and one work node. To demonstrate the workflow, I will use `kubernetes-with-raw-docker:v1.19.8`. You can get the same image by executing `sealer pull kubernetes-with-raw-docker:v1.19.8`.

#### Step 2: get the kyverno install yaml

Download the install yaml of kyverno at `https://raw.githubusercontent.com/kyverno/kyverno/release-1.5/definitions/release/install.yaml`, you can replace the verion to what you want. I use 1.5 in this demonstration.

#### Step 3: create a ClusterPolicy

Create a yaml with the following content:

```yaml
apiVersion : kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: redirect-registry
spec:
  background: false
  rules:
  - name: prepend-registry-containers
    match:
      resources:
        kinds:
        - Pod
    preconditions:
      all:
      - key: "{{request.operation}}"
        operator: In
        value:
        - CREATE
        - UPDATE
    mutate:
      foreach:
      - list: "request.object.spec.containers"
        patchStrategicMerge:
          spec:
            containers:
            - name: "{{ element.name }}"
              image: "sea.hub:5000/library/{{ images.containers.{{element.name}}.path}}:{{images.containers.{{element.name}}.tag}}"
  - name: prepend-registry-initcontainers
    match:
      resources:
        kinds:
        - Pod
    preconditions:
      all:
      - key: "{{request.operation}}"
        operator: In
        value:
        - CREATE
        - UPDATE
    mutate:
      foreach:
      - list: "request.object.spec.initContainers"
        patchStrategicMerge:
          spec:
            initContainers:
            - name: "{{ element.name }}"
              image: "sea.hub:5000/library/{{ images.initContainers.{{element.name}}.path}}:{{images.initContainers.{{element.name}}.tag}}"

```

This ClusterPolicy will redirect image pull request to private registry `sea.hub:5000`, and I name this file as redirect-registry.yaml

#### Step 4: create a shell script to monitor kyverno pod

Because the state of kyverno pod should be running, then the ClusterPolicy will work. It's advised to create and run the following shell script to monitor the state of kyverno pod until it become running.

```shell
#!/bin/bash

echo "[kyverno-start]: Waiting for the kyverno to be ready"

while true
do
  podstatus=`kubectl get pod -n kyverno -o go-template --template={{range.items}}.{{.status.phase}}{{end}}`
  if [ "$podstatus" == "Running" ];then
    break
  fi
  sleep 5
done

echo "kyverno is running"
```

I named this file `wait-kyverno-ready.sh`.

#### Step 5: create the build content

Create a directory with three files: the install.yaml in step 2, redirect-registry.yaml in step 3 and a Kubefile whose content is following:

```shell
FROM kubernetes-with-raw-docker:v1.19.8
COPY . .
CMD kubectl create -f install.yaml && kubectl create -f redirect-registry.yaml
CMD bash wait-kyverno-ready.sh
```

#### Step 6: build the image

Supposing you are at the directory create at step 4, please execute `sealer build --mode lite -t <image-name:image:tag> .`