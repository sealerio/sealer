## sealer run

run a cluster with images and arguments

### Synopsis

sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters [arg] --nodes [arg]

```
sealer run [flags]
```

### Examples

```

create default cluster:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9

create cluster by cloud provider, just set the number of masters or nodes:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters 3 --nodes 3

create cluster to your baremetal server, appoint the iplist:
	sealer run registry.cn-qingdao.aliyuncs.com/sealer-io/kubernetes:v1.19.9 --masters 192.168.0.2,192.168.0.3,192.168.0.4 \
		--nodes 192.168.0.5,192.168.0.6,192.168.0.7

```

### Options

```
  -h, --help               help for run
  -m, --masters string     set Count or IPList to masters
  -n, --nodes string       set Count or IPList to nodes
  -p, --passwd string      set cloud provider or baremetal server password
      --pk string          set baremetal server private key (default "/Users/sunzhiheng/.ssh/id_rsa")
      --pk-passwd string   set baremetal server  private key password
      --podcidr string     set default pod CIDR network. example '10.233.0.0/18'
      --svccidr string     set default service CIDR network. example '10.233.64.0/18'
  -u, --user string        set baremetal server username (default "root")
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

