# auto-build

Automatic cluster image building is to meet the needs of some users. Users need to build a specified version of kubernetes for automatic building.

## Quick start
Standard directive:

```shell
/imagebuild + version + arch
```
`/imagebuild` : To trigger an automated build.

`version` : The version corresponding to kubernetes.

`arch`: For example, arm64 or AMD64 or arm.

Note: none of the three conditions is indispensable

Take the image of version 1.20.14 as an example. You don't need to pay attention to other operations. You just need to comment on the standard trigger instruction in the issue.

```shell
/imagebuild 1.20.14 amd64
```

After inputting, GitHub action will start to execute the automatic construction of the basic image of the corresponding version of kubernetes.

When `kubernetes: version auto` appears in the comment area, it indicates that the image construction is completed.
