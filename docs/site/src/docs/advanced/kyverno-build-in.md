# Kyverno build-in

## Motivations

It's common that some k8s clusters have their own private image registry,and they don't want to pull images from other registry for safety reasons.this page is about how to introduce kyverno in k8s cluster,which will redirect image pull request to Specified registry.

## Uses case

### How to use it

we provide a out-of-the-box cloud image which introduces kyverno into cluster:`kubernetes-raw_docerk-kyverno:v1.19.8`.Note that it contains no docker images other than those necessary to run a k8s cluster,so if you want use this cloud image and you also need other docker images(such as `nginx`) to run a container,you need to cache the docker image to your private registry.

Of course `sealer` can help you do this,use `nginx` as an example.
Firstly edit a Kubefile with the following content:

```
FROM kubernetes-raw_docerk-kyverno:v1.19.8
COPY imageList manifests
CMD kubectl run nginx --image=nginx:latest
```

Secondly include nginx in the file `imageList`.
You can execute `cat imageList` to make sure you have did this,and the result may seem like this:

```
 [root@ubuntu ~]# cat imageList
 nginx:latest
```

Thirdly execute `sealer build` to build a new cloud image

```
 [root@ubuntu ~]# sealer build -t my-nginx-kubernetes:v1.19.8 .
```

just a simple command and let sealer help you cache `nginx:latest` image to private registry.

Now you can use this new cloud image to create k8s cluster.After your cluster startup,there is already a pod running `nginx:latest` image,you can see it by execute `kubectl describe pod nginx`.And you can also create more pods running `nginx:latest` image

### How to get it

the following is a sequence steps of building kyverno build-in cloud image

#### step 1：choose a base image

choose a base image which can create a k8s cluster with at least one master node and one work node.To demonstrate the workflow,I will use `kubernetes-with-raw-docker:v1.19.8`.you can get the same image by executing `sealer pull kubernetes-with-raw-docker:v1.19.8`.

#### step 2:get the kyverno install yaml

download the install yaml of kyverno at `https://raw.githubusercontent.com/kyverno/kyverno/release-1.5/definitions/release/install.yaml`,you can replace the verion to what you want.I use 1.5 in this demonstration.

#### step 3:create a ClusterPolicy

create a yaml with the following content:

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
              image: "sea.hub:5000/{{ images.containers.{{element.name}}.path}}:{{images.containers.{{element.name}}.tag}}"
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
              image: "sea.hub:5000/{{ images.initContainers.{{element.name}}.path}}:{{images.initContainers.{{element.name}}.tag}}"

```

this ClusterPolicy will redirect image pull request to private registry `sea.hub:5000`,and I name this file as redirect-registry.yaml

#### step 4:create the build content

create a directory with three files:the install.yaml in step 2、redirect-registry.yaml in step 3 and a Kubefile whose content is following:

```shell
FROM kubernetes-with-raw-docker:v1.19.8
COPY . .
CMD kubectl create -f install.yaml && kubectl create -f redirect-registry.yaml
```

#### step 5:build the image

Supposing you are at the directory create at step 4.execute `sealer build --mode lite -t <image-name:image:tag> .`