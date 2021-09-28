## sealer delete

delete a cluster

### Synopsis

if provider is BARESERVER will delete kubernetes nodes or IPList;  if provider is ALI_CLOUD, will delete all the infra resources or count

```
sealer delete [flags]
```

### Examples

```

delete to default cluster: 
	sealer delete --masters x.x.x.x --nodes x.x.x.x
	sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
delete to cluster by cloud provider, just set the number of masters or nodes:
	sealer delete --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
	sealer delete --masters 2 --nodes 3 -f /root/.sealer/specify-cluster/Clusterfile
delete all:
	sealer delete --all [--force]
	sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]

```

### Options

```
  -f, --Clusterfile string   delete a kubernetes cluster with Clusterfile Annotations
  -a, --all                  this flags is for delete nodes, if this is true, empty all node ip
      --force                We also can input an --force flag to delete cluster by force
  -h, --help                 help for delete
  -m, --masters string       reduce Count or IPList to masters
  -n, --nodes string         reduce Count or IPList to nodes
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

