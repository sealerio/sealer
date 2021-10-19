+++
title = "ARM CloudImage"
description = "Ship kubernetes on ARM machines."
date = 2021-05-01T19:30:00+00:00
updated = 2021-05-01T19:30:00+00:00
draft = false
weight = 30
sort_by = "weight"
template = "docs/page.html"

[extra]
lead = "Using ARM CloudImage"
toc = true
top = false
+++

# Download sealer

For example download v0.5.0:

```shell script
wget https://github.com/alibaba/sealer/releases/download/v0.5.0/sealer-v0.5.0-linux-arm64.tar.gz
```

# Run a cluster on ARM platform

Just using the ARM CloudImage `kubernetes-arm64:v1.19.7`:

```shell script
sealer run kubernetes-arm64:v1.19.7 --master 192.168.0.3 --passwd xxx
```