This CloudImage only contains a debug:ubuntu docker image. You can use this CloudImage as base image to build other CloudImages.

# Build a kubernetes-debug-ubuntu CloudImage

```
sealer build -m lite -t kubernetes-debug-ubuntu:v1.19.9 .
sealer push kubernetes-debug-ubuntu:v1.19.9
```

# Using kubernetes-debug-ubuntu CloudImage as Base Image

```
FROM kubernetes-debug-ubuntu:v1.19.9
RUN wget https://raw.githubusercontent.com/kubernetes/dashboard/v2.2.0/aio/deploy/recommended.yaml
CMD kubectl apply -f recommended.yaml
```

## What can this CloudImage do

This CloudImage contains a debug:ubuntu docker image which may needed by `sealer debug`.  You can see  [sealer debug document](../../docs/debug/README.md) to see how to use `sealer debug`.

