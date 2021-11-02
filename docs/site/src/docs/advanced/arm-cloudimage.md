# ARM CloudImage

Download sealer for example download v0.5.0:

```shell script
wget https://github.com/alibaba/sealer/releases/download/v0.5.0/sealer-v0.5.0-linux-arm64.tar.gz
```

# Run a cluster on ARM platform

Just using the ARM CloudImage `kubernetes-arm64:v1.19.7`:

```shell script
sealer run kubernetes-arm64:v1.19.7 --master 192.168.0.3 --passwd xxx
```
