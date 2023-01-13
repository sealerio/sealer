# Global configuration

The feature of global configuration is to expose the parameters of distributed applications in the entire cluster mirror.
It is highly recommended exposing only a few parameters that users need to care about.

If too many parameters need to be exposed, for example, the entire helm's values ​​want to be exposed,
then it is recommended to build a new image and put the configuration in to overwrite it.

Using dashboard as an example, we made a cluster mirror of dashboard,
but different users want to use different port numbers while installing.
In this scenario, sealer provides a way to expose this port number parameter to the environment
variable of Clusterfile.

Use global configuration capabilities
For the image builder, this parameter needs to be extracted when making the image.
Take the yaml of the dashboard as an example:

dashboard.yaml.tmpl:

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
      targetPort: {{ .DashBoardPort }}
  selector:
    k8s-app: kubernetes-dashboard
...
```

To write kubefile, you need to copy yaml to the "manifests" directory at this time,
sealer only renders the files in this directory:

sealer will render the .tmpl file and create a new file named `dashboard.yaml`

```yaml
FROM kubernetes:1.16.9
COPY dashobard.yaml.tmpl manifests/ # only support render template files in `manifests etc charts` dirs
CMD kubectl apply -f manifests/dashobard.yaml
```

For users, they only need to specify the cluster environment variables:

```shell script
sealer run -e DashBoardPort=8443 mydashboard:latest -m xxx -n xxx -p xxx
```

Or specify in Clusterfile:

```yaml
apiVersion: sealer.io/v1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: mydashobard:latest
  env:
    DashBoardPort=6443 # Specify a custom port here, which will be rendered into the mirrored yaml
...
```

## Using Env in shell plugin or other scripts

[Using env in scripts](https://github.com/sealerio/sealer/blob/main/docs/design/clusterfile-v2.md#using-env-in-configs-and-script)

## Application config

Application config file:

Clusterfile:

```
apiVersion: sealer.io/v1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-app/my-SAAS-all-inone:latest
  provider: BAREMETAL
---
apiVersion: sealer.io/v1
kind: Config
metadata:
  name: mysql-config
spec:
  path: etc/mysql.yaml
  data: |
       mysql-user: root
       mysql-passwd: xxx
...
---
apiVersion: sealer.io/v1
kind: Config
metadata:
  name: redis-config
spec:
  path: etc/redis.yaml
  data: |
       redis-user: root
       redis-passwd: xxx
...
```

When apply this Clusterfile, sealer will generate some values file for application config. Named etc/mysql-config.yaml  etc/redis-config.yaml.

So if you want to use this config, Kubefile is like this:

```
FROM kuberentes:v1.19.9
...
CMD helm install mysql -f etc/mysql-config.yaml
CMD helm install mysql -f etc/redis-config.yaml
```

## Development Document

Before mounting Rootfs, templates need to be rendered for the files in etc, charts, and manifest directories,
and render environment variables and annotations to the [configuration file].
Generate the global.yaml file to the etc directory