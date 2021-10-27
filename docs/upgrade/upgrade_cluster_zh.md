# sealer upgrade 命令的原理与实现

```
截止PR#783，upgrade命令的工作流程可以概括如下：
解析命令的参数->执行Apply()函数->执行diff()函数得到todoList->执行todoList中的函数->升级成功->更新当前集群的Image变量值。
```

**接下来对流程中的每一步进行详细分析**
> 1.解析命令参数，根据参数创建applier。解析命令的参数值，主要是获取当前集群的Clusterfile和期望状态的镜像。根据Clusterfile得到一个`v1.Cluster`对象，由于集群的期望状态与当前状态的区别只是使用不同的镜像，故此步可以建立一个表示集群期待状态的`v1.Cluster`对象，然后使用改`v1.Cluster`对象做为参数，调用apply包下的`NewApplier()`函数，返回一个满足`apply.Interface`接口的applier对象。
---
> 2.执行`Apply()`函数。执行步骤1中得到的applier的`apply()`函数。applier有两个变量成员`ClusterDesired`和`ClusterCurrent`。到此时为止，`ClusterDesired`已经根据步骤1中`NewApplier`函数的参数`v1.Cluster`进行了初始化，而`ClusterCurrent`为空。为了接下来能够进行期望状态与当前状态的镜像对比，需要找出当前状态的镜像，代码如下：

```
clusterFilePath := common.GetClusterWorkClusterfile(c.ClusterDesired.Name)
if utils.IsFileExist(clusterFilePath) {
	clusterBeforeUpgrade, err := GetClusterFromFile(clusterFilePath)
	if err != nil {
		return err
	}
	if clusterBeforeUpgrade.Spec.Image != c.ClusterDesired.Spec.Image {
		c.ClusterCurrent.Spec.Image = clusterBeforeUpgrade.Spec.Image
	}
}
```

> 此段代码所做的事情是：到指定路径下找到Clusterfile文件，然后根据Clusterfile文件得到当前集群的状态（此时的状态是未升级的状态），与期望状态的镜像进行对比。如果不同，则说明需要进行升级操作，设置`ClusterCurrent`的`Image`属性，便于接下来进行对比。
---
> 3.执行`diff()`函数，把需要执行的函数加入todoList。在`diff()`函数中进行对比，判断是否需要进行升级。如果需要进行升级，则在todoList中加入相应的函数。

```
if c.ClusterDesired.Spec.Image != c.ClusterCurrent.Spec.Image {
	todoList = append(todoList, MountRootfs)
	todoList = append(todoList, Upgrade)
}
```

> 此处要说明的是：为了执行一个完整的upgrade操作，在该`if`语句块之前还需要加入todoList的函数有：`PullIfNotExist`和`MountImage`，在该`if`语句块之后还需要加入todoList的函数有：`Guest`和`UnMountImage`。
---
> 4.执行todoList中的函数。此处着重介绍`Upgrade`函数的原理，其他函数的介绍参考相应的文档。`Upgrade`函数的具体实现在`runtime`包下，upgrade.go文件。流程可以概括为：==创建ssh.Client对象->升级主控制节点->升级其余控制节点->升级工作节点==。创建ssh.Client对象的代码如下，其中参数的类型是`*v1.Cluster`

```
client, err := ssh.NewSSHClientWithCluster(cluster)
```

> 所有节点的升级原理大致相同，核心是通过IP地址登录到对应节点，执行集群升级命令。以升级主控制节点为例，主要做了以下几件事情：
- 将节点rootfs文件系统下bin目录中的二进制文件赋予执行权限，并移动至/usr/bin目录下。（因为/usr/bin包含在环境变量PATH的值中）
- 用kubectl drain排空节点。
- 用kubectl upgrade升级节点。
- 重启节点上的kubelet。
- 将节点标记为可调度。
>其余控制节点上的操作与主控制节点一致，工作节点上不用进行“排空节点”的操作和“标记节点为可调度”的操作。
---
> 最后，在upgrade操作成功之后，将`ClusterCurrent`的`Image`属性值，更新为`ClusterDesired`的`Image`属性值。


