# Multi-arch build

## Build cloud image

Kubefile:

```shell
FROM kubernetes:v1.19.8
COPY dashborad.yaml manifests
COPY ${ARCH}/helm bin # copy binary file,make sure the build context have the same number platform binary files.
COPY my-mysql charts
CMD helm install my-mysql bitnami/mysql --version 8.8.26
CMD kubectl apply -f manifests/dashborad.yaml
```

sealer build cmd line:

```shell
sealer build --platform linux/arm64/v8,linux/amd64 -t kubernetes-multi-arch:v1.19.8
```

### Cloud image manifests list

```json
{
  "kubernetes-multi-arch:v1.19.8": {
    "manifests": [
      {
        "id": "52c3b10849c852649e66c2f7ed531f05bd97586ab61fa2cc82b4e79d80484b82",
        "created": "2022-03-08T14:23:18.571666683+08:00",
        "size": 826082517,
        "platform": {
          "architecture": "amd64",
          "os": "linux"
        }
      },
      {
        "id": "9e596d0a54177f29093236f65a9c6098590c67ea8b0dde4e09a5a49124cec7d0",
        "created": "2022-03-08T14:23:18.571666683+08:00",
        "size": 826082517,
        "platform": {
          "architecture": "arm64",
          "os": "linux",
          "variant": "v8"
        }
      }
    ]
  }
}
```

### Cloud image manifests

#### amd64 cloud image

`cat 52c3b10849c852649e66c2f7ed531f05bd97586ab61fa2cc82b4e79d80484b82.yaml`

```yaml
kind: Image
metadata:
  annotations:
    sea.aliyun.com/ClusterFile: |
      apiVersion: sealer.cloud/v2
      kind: Cluster
      metadata:
        creationTimestamp: null
        name: my-cluster
      spec:
        image: kubernetes-multi-arch:v1.19.8
        ssh:
          port: "22"
          user: root
      status: {}
  creationTimestamp: null
  name: kubernetes-multi-arch:v1.19.8
spec:
  id: 52c3b10849c852649e66c2f7ed531f05bd97586ab61fa2cc82b4e79d80484b82
  image_config:
    cmd:
      current:
        - helm install my-mysql bitnami/mysql --version 8.8.26
        - kubectl apply -f manifests/dashborad.yaml
  layers:
    - id: sha256:ba2221cfa297f483e195fd61b30651f2426303965f8f1dc69cf5d4eff635af9a
      type: COPY
      value: . .
    - id: sha256:e00f3ade42dc8cebffa2314b8bee4ee5472c5a915c2ba3687a588d47657b3d6a
      type: COPY
      value: dashborad.yaml manifests
    - id: sha256:5cd1d3347ba4d9a0edea555da8489633f73a42266e33cc8c55245b8791c6ff72
      type: COPY
      value: my-mysql charts
    - id: sha256:4f782a7c667b104f59140aa7836af9138836eef971764a426c309df4f9334ac6
      type: BASE ## only amd64 docker images
      value: rootfs cache
  platform:
    architecture: amd64
    os: linux
  sealer_version: latest
status: { }
```

#### arm64 v8 cloud image

`cat 9e596d0a54177f29093236f65a9c6098590c67ea8b0dde4e09a5a49124cec7d0.yaml`

```yaml
kind: Image
metadata:
  annotations:
    sea.aliyun.com/ClusterFile: |
      apiVersion: sealer.cloud/v2
      kind: Cluster
      metadata:
        creationTimestamp: null
        name: my-cluster
      spec:
        image: kubernetes-multi-arch:v1.19.8
        ssh:
          port: "22"
          user: root
      status: {}
  creationTimestamp: null
  name: kubernetes-multi-arch:v1.19.8
spec:
  id: 9e596d0a54177f29093236f65a9c6098590c67ea8b0dde4e09a5a49124cec7d0
  image_config:
    cmd:
      current:
        - helm install my-mysql bitnami/mysql --version 8.8.26
        - kubectl apply -f manifests/dashborad.yaml
  layers:
    - id: sha256:ba2221cfa297f483e195fd61b30651f2426303965f8f1dc69cf5d4eff635af9a
      type: COPY
      value: . .
    - id: sha256:e00f3ade42dc8cebffa2314b8bee4ee5472c5a915c2ba3687a588d47657b3d6a
      type: COPY
      value: dashborad.yaml manifests
    - id: sha256:5cd1d3347ba4d9a0edea555da8489633f73a42266e33cc8c55245b8791c6ff72
      type: COPY
      value: my-mysql charts
    - id: sha256:11c980114032d5f583c3861f1077bcc2f6d4e0e38b15219205fe22de044fd3a5
      type: BASE ## only save arm64 v8 docker images
      value: rootfs cache
  platform:
    architecture: arm64
    os: linux
    variant: v8
  sealer_version: latest
status: { }
```

## Run cloud image

| IP      | Platform | OS    |
| :---        |    :----:   |          ---: |
| 192.168.1.1      | amd64       | linux  |
| 192.168.1.2   | arm64        | linux      |

sealer run cmd line:

```shell
sealer run -m 192.168.1.1 -n 192.168.1.2 kubernetes-multi-arch:v1.19.8
```

### Mount image

we have three mounter point:

1. amdMounter : lower layers include base amd rootfs and all image data.
2. armMounter : lower layers include base arm rootfs and all image data.
3. registryMounter : ${amdMounter}/registry + ${armMounter}/registry.

### Mount rootfs

1. For master:only have amdMounter data
2. For node :only have armMounter data

## Save cloud image

if not specify the platform will save them all. save two image_metadata.yaml and all manifests file.

If you want save amd64 images of kubernetes-multi-arch:v1.19.8 using platform arg

`sealer save -o kubernetes.tar kubernetes-multi-arch:v1.19.8 --platform linux/amd64`

manifests file and one image_metadata.yaml :

```json
{
  "kubernetes-multi-arch:v1.19.8": {
    "manifests": [
      {
        "id": "52c3b10849c852649e66c2f7ed531f05bd97586ab61fa2cc82b4e79d80484b82",
        "created": "2022-03-08T14:23:18.571666683+08:00",
        "size": 826082517,
        "platform": {
          "architecture": "amd64",
          "os": "linux"
        }
      }
    ]
  }
}
```

## Load cloud image

`sealer load -i kubernetes.tar`

## Inspect cloud image

`sealer inspect kubernetes-multi-arch:v1.19.8 --platform linux/amd64`

## Delete cloud image

if not specify the platform will delete them all. If you only want to delete amd64 images of `kubernetes-multi-arch:
v1.19.8`.

`sealer delete kubernetes-multi-arch:v1.19.8 --platform linux/amd64`

## Pull cloud image

`sealer pull kubernetes-multi-arch:v1.19.8 --platform linux/amd64`

## Push cloud image

`sealer push kubernetes-multi-arch:v1.19.8 --platform linux/amd64`