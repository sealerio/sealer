# Kyverno build-in
## step 1：choose a base image
choose a base image which can create a k8s cluster with at least one master node and one work node.To demonstrate the workflow,I will use `kubernetes:v1.19.8`.you can get the same image by executing `sealer pull kubernetes:v1.19.8`.
## step 2:get the kyverno install yaml
download the install yaml of kyverno at `https://raw.githubusercontent.com/kyverno/kyverno/release-1.5/definitions/release/install.yaml`,you can replace the verion to what you want.I use 1.5 in this demonstration.
## step 3:create a ClusterPolicy
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

this ClusterPolicy will redirect image pull request to my private registry `sea.hub:5000`,and I name it as redirect-registry.yaml
## step 4:create the build content
create a directory with three files:the install.yaml in step 2、redirect-registry.yaml in step 3 and a Kubefile whose content is following:

```shell
FROM kubernetes:v1.19.8
COPY . .
RUN kubectl create -f install.yaml && kubectl create -f redirect-registry.yaml
```

## step 5:build the image
Supposing you are at the directory create at step 4.execute `sealer build -t <image-name:image:tag> .`