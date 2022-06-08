## sealer delete

delete an existing cluster

### Synopsis

delete command is used to delete part or all of existing cluster.
User can delete cluster by explicitly specifying node IP, Clusterfile, or cluster name.

```
sealer delete [flags]
```

### Examples

```

delete default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
	sealer delete -c my-cluster [--force]

```

### Options

```
  -f, --Clusterfile string   delete a kubernetes cluster with Clusterfile Annotations
  -a, --all                  this flags is for delete nodes, if this is true, empty all node ip
  -c, --cluster string       delete a kubernetes cluster with cluster name
      --force                We also can input an --force flag to delete cluster by force
  -h, --help                 help for delete
  -m, --masters string       reduce Count or IPList to masters
  -n, --nodes string         reduce Count or IPList to nodes
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

