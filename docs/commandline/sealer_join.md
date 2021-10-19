## sealer join

join node to cluster

```
sealer join [flags]
```

### Examples

```

join to default cluster: merge
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
join to cluster by cloud provider, just set the number of masters or nodes:
	sealer join --masters 2 --nodes 3
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer join --masters 2 --nodes 3 -c my-cluster

```

### Options

```
  -c, --cluster-name string   submit one cluster name
  -h, --help                  help for join
  -m, --masters string        set Count or IPList to masters
  -n, --nodes string          set Count or IPList to nodes
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

