# Build a kubernetes-withflannel CloudImage

```shell script
sealer build -b lite -t kubernetes-withflannel:v1.19.9 .
sealer push kubernetes-withcalico:v1.19.9
```

## cni-plugin version

+ 0.8.3 https://github.com/containernetworking/plugins/releases/download/v0.8.3/cni-plugins-linux-amd64-v0.8.3.tgz

## flannel version
+ v0.14.0 https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml

## How to use it
```
    sealer pull  registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9
    sealer run registry-1.docker.io/bryantrh/kubernetes-withflannel:v1.19.9 --masters xxxx --nodes xxxx
```

##  How to rebuild it
First create the directory cni ，then download cni-plugin and tar
```
cni
├── bandwidth
├── bridge
├── dhcp
├── firewall
├── flannel
├── host-device
├── host-local
├── ipvlan
├── loopback
├── macvlan
├── portmap
├── ptp
├── sbr
├── static
├── tuning
└── vlan
```

second change init-kube.sh
```
...

#cni
mkdir /opt/cni/bin -p
chmod -R 755 ../cni/*
chmod 644 ../cni
cp ../cni/* /opt/cni/bin

...
```


```shell script
FROM kubernetes:v1.19.9-alpine
#RUN wget https://github.com/containernetworking/plugins/releases/download/v0.8.3/cni-plugins-linux-amd64-v0.8.3.tgz 
COPY cni .
COPY init-kube.sh /scripts/
COPY kube-flannel.yml manifests/
CMD kubectl apply -f manifests/kube-flannel.yml
```
