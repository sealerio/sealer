# ARM CloudImage

Download sealer for example download v0.8.5:

```shell script
wget https://github.com/sealerio/sealer/releases/download/v0.8.5/sealer-v0.8.5-linux-arm64.tar.gz
```

## Run a cluster on ARM platform

```shell script
sealer run kubernetes:v1.19.8 --master 192.168.0.3 --passwd xxx
```

## Build an ARM cloud image

Just "FROM" the ARM cloud image to run sealer build on any platform will build out the ARM cloud image .

Kubefile example:

```shell
FROM kubernetes-arm64:v1.19.7
COPY imageList manifests
COPY recommended.yaml manifests
CMD kubectl apply -f manifests/recommended.yaml
```

Run an ARM dashboard cloud image.

```shell
sealer build -f Kubefile -t my-dashboard:v1 .
```

Run this arm dashboard cloud image.

```shell
sealer run my-dashboard:v1 --master 192.168.0.3 --passwd xxx
```