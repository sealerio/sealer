# Auto-build

Automatic ClusterImage build is to meet the needs of some users, Users need to build a specified version of kubernetes for automatic building.

## Quick start
Standard directive:

```shell
/imagebuild + version + arch
```

`/imagebuild` : Trigger an automated build.

`version` : Version corresponding to kubernetes.

`arch`: Input version is AMD64 or arm64.

Note: none of the three conditions is indispensable

You don't need to pay attention to other operations. You just need to comment on the standard trigger instruction in the issue.

Take the image of version 1.20.14 as an example.

input:

```shell
/imagebuild 1.20.14 amd64
```

When `Image built successfully : kubernetes: version` appears in the comment area, it indicates that the image build is completed.

Image name:

```shell
kubernetes:v1.20.14
```

