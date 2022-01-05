# 全局配置

全局配置的特性是为了暴露出整个集群镜像里面分布式应用的参数，非常推荐的是只暴露出少量用户需要关心的参数。

如果需要暴露太多参数的话，比如整个helm的values都希望暴露出来，那更建议build一个新镜像把配置放进去覆盖。

以dashboard为例，我们制作了一个dashboard的集群镜像，但是不同用户在进行安装时希望使用不同的端口号，这种场景sealer提供
一种方式把这个端口号参数透出到Clusterfile的环境变量中去。

## 使用全局配置能力

对于镜像的构建者在制作镜像时需要把这个参数抽离出来，以dashboard的yaml为例：

dashboard.yaml:

```yaml
...
kind: Service
apiVersion: v1
metadata:
  labels:
    k8s-app: kubernetes-dashboard
  name: kubernetes-dashboard
  namespace: kubernetes-dashboard
spec:
  ports:
    - port: 443
      targetPort: {{ DashBoardPort }}
  selector:
    k8s-app: kubernetes-dashboard
...
```

编写kubefile,此时需要把yaml拷贝到manifests目录，sealer仅对这个目录下的文件进行渲染:

```shell script
FROM kubernetes:1.16.9
COPY dashobard.yaml manifests/
CMD kubectl apply -f manifests/dashobard.yaml
```

对于使用者只需要指定集群环境变量即可：

```shell script
sealer run -e DashBoardPort=8443 mydashboard:latest -m xxx -n xxx -p xxx
```

或者在Clusterfile中指定：

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: mydashobard:latest
  provider: BAREMETAL
  env:
    DashBoardPort: 6443 # 在这里指定自定义端口, 会被渲染到镜像的yaml中
  ssh:
    passwd:
    pk: xxx
...
```

## 与helm配合使用

sealer在运行时同样会生成一个非常全的Clusterfile文件到etc目录下，意味着helm chart中是可以通过一定的方法获取到这些参数的。

dashboard的chart values就可以这样写：

```yaml
spec:
  env:
    DashboardPort: 6443
```

Kubefile:

```yaml
FROM kubernetes:v1.16.9
COPY dashboard-chart .
CMD helm install dashboard dashboard-chart -f etc/global.yaml
```

这样global.yaml里面的值就会覆盖掉dashboard中的默认端口参数。

## 开发文档

1. 在apply guest之前对manifest目录下的文件进行模板渲染，把环境变量和annotations渲染到[配置文件中](
   https://github.com/alibaba/sealer/blob/main/pkg/guest/guest.go#L28), guest模块就是去处理Kubefile中RUN CMD这类指令的。
2. 生成global.yaml文件到etc目录下。
