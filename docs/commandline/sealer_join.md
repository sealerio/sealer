## sealer join

join new master or worker node to specified cluster

```
sealer join [flags]
```

### Examples

```

join default cluster:
	sealer join --masters x.x.x.x --nodes x.x.x.x
    sealer join --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y

```

### Options

```
  -c, --cluster-name string   specify the name of cluster
  -h, --help                  help for join
  -m, --masters string        set Count or IPList to masters
  -n, --nodes string          set Count or IPList to nodes
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

