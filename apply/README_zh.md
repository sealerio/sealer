# apply集群镜像

apply模块是一个顶层的封装模块，负责让集群实例保持clusterfile中定义的终态。

```
Apply(Clusterfile)
  image.Load()
  fs.Mount()
  runtime.Run()
    Init()
    Hook()
    JoinMasters()
    JoinNodes()
    StaticPod()
  guest.Apply()
```
