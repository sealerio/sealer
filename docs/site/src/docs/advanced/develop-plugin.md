# Develop out of tree plugin.

## Motivations

Sealer support common plugins such as hostname plugin,label plugin,which is build in,user could define and use it
according their requests. Sealer also support to load out of tree plugin which is written by golang. This page is about
how to extend the new plugin type and how to develop an out of tree plugin.

## Uses case

### How to develop an out of tree plugin

if user doesn't want their plugin code to be open sourced, we can develop an out of tree plugin to use it.

1. implement the golang plugin interface and expose the variable named `Plugin`.

* package name must be "main"
* exposed variable must be "Plugin"

Examples:list_nodes.go

```shell
package main

import (
	"fmt"
	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/plugin"
)

type list string

func (l *list) GetPluginType() string {
	return "LIST_NODE"
}

func (l *list) Run(context plugin.Context, phase plugin.Phase) error {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	nodeList, err := client.ListNodes()
	if err != nil {
		return fmt.Errorf("cluster nodes not found, %v", err)
	}
	for _, v := range nodeList.Items {
		fmt.Println(v.Name)
	}
	return nil
}

var Plugin list
```

2. build the new plugin as so file. plugin file and sealer source code must in the same golang runtime in order to avoid
   compilation problems. we suggest the so file must build with the specific sealer version you used. otherwise,sealer
   will fail to load the so file. you can replace the build file at the test directory
   under [Example](https://github.com/alibaba/sealer/blob/main/plugin) to build your own so file.

```shell
go build -buildmode=plugin -o list_nodes.so list_nodes.go
```

3. use the new so file

Copy the so file and plugin config file to your cloud image.We can also append plugin yaml to Clusterfile and
use `sealer apply -f Clusterfile` to test it.

Kubefile:

```shell
FROM kubernetes:v1.19.8
COPY list_nodes.so plugin
COPY list_nodes.yaml plugin
```

```shell script
sealer build -m lite -t kubernetes-post-install:v1.19.8 .
```

list_nodes.yaml:

```yaml
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: list_nodes.so # out of tree plugin name
spec:
  type: TEST_SO # define your own plugin type.
  action: PostInstall # which stage will this plugin be applied.
```

apply it in your cluster: `sealer run kubernetes-post-install:v1.19.8 -m x.x.x.x -p xxx`