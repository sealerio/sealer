# Build a kubernetes-withflannel CloudImage

```shell script
sealer build -b lite -t registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9 .
sealer push registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9
```

## cni-plugin version

+ 0.8.3 <https://github.com/containernetworking/plugins/releases/download/v0.8.3/cni-plugins-linux-amd64-v0.8.3.tgz>

## flannel version

+ v0.14.0 <https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml>

## How to use it

```shell script
#Pull image
sealer pull  registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9
#Run it
sealer run registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9 --masters xxxx --nodes xxxx
```

## How to rebuild it
first change init-kube.sh

```shell script
...
#cni
mkdir /opt/cni/bin -p
chmod -R 755 ../cni/*
chmod 644 ../cni
cp ../cni/* /opt/cni/bin
...

```

second  create Kubefile

```shell script
FROM kubernetes:v1.19.9-alpine
RUN wget https://github.com/containernetworking/plugins/releases/download/v0.8.3/cni-plugins-linux-amd64-v0.8.3.tgz && mkdir cni && tar -xf cni-plugins-linux-amd64-v0.8.3.tgz -C cni/
#COPY cni .
COPY init-kube.sh /scripts/
COPY kube-flannel.yml manifests/
CMD kubectl apply -f manifests/kube-flannel.yml
```
