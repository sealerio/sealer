## sealer gen

generate a Clusterfile to take over a normal cluster which is not deployed by sealer

### Synopsis

sealer gen --passwd xxxx --image kubernetes:v1.19.8

The takeover actually is to generate a Clusterfile by kubeconfig.
Sealer will call kubernetes API to get masters and nodes IP info, then generate a Clusterfile.
Also sealer will pull a ClusterImage which matches the kubernetes version.

Check generated Clusterfile: 'cat .sealer/<cluster name>/Clusterfile'

The master should has 'node-role.kubernetes.io/master' label.

Then you can use any sealer command to manage the cluster like:

> Upgrade cluster
	sealer upgrade --image kubernetes:v1.22.0

> Scale
	sealer join --node x.x.x.x

> Deploy a ClusterImage into the cluster
	sealer run mysql-cluster:5.8

```
sealer gen [flags]
```

### Options

```
  -h, --help               help for gen
      --image string       Set taken over ClusterImage
      --name string        Set taken over cluster name (default "default")
      --passwd string      Set taken over ssh passwd
      --pk string          set server private key (default "/root/.ssh/id_rsa")
      --pk-passwd string   set server private key password
      --port uint16        set the sshd service port number for the server (default port: 22) (default 22)
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

