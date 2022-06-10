## sealer run

start to run a cluster from a ClusterImage

### Synopsis

sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.8 --masters [arg] --nodes [arg]

```
sealer run [flags]
```

### Examples

```

create cluster to your bare metal server, appoint the iplist:
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
		--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx

specify server SSH port :
  All servers use the same SSH port (default port: 22):
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
	--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --port 24 --passwd xxx

  Different SSH port numbers exist：
	sealer run kubernetes:v1.19.8 --masters 192.168.0.2,192.168.0.3:23,192.168.0.4:24 \
	--nodes 192.168.0.5:25,192.168.0.6:25,192.168.0.7:27 --passwd xxx

create a cluster with custom environment variables:
	sealer run -e DashBoardPort=8443 mydashboard:latest  --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
	--nodes 192.168.0.5,192.168.0.6,192.168.0.7 --passwd xxx

```

### Options

```
      --cluster-name string   set cluster name (default "my-cluster")
      --cmd-args strings      set args for image cmd instruction
  -e, --env strings           set custom environment variables
  -h, --help                  help for run
  -m, --masters string        set count or IPList to masters
  -n, --nodes string          set count or IPList to nodes
  -p, --passwd string         set cloud provider or baremetal server password
      --pk string             set baremetal server private key (default "/root/.ssh/id_rsa")
      --pk-passwd string      set baremetal server private key password
      --port uint16           set the sshd service port number for the server (default port: 22) (default 22)
      --provider ALI_CLOUD    set infra provider, example ALI_CLOUD, the local server need ignore this
  -u, --user string           set baremetal server username (default "root")
```

### Options inherited from parent commands

```
      --config string   config file of sealer tool (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
      --hide-path       hide the log path
      --hide-time       hide the log time
```

### SEE ALSO

* [sealer](sealer.md)	 - A tool to build, share and run any distributed applications.

