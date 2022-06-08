## sealer apply

apply a Kubernetes cluster via specified Clusterfile

### Synopsis

apply command is used to apply a Kubernetes cluster via specified Clusterfile.
If the Clusterfile is applied first time, Kubernetes cluster will be created. Otherwise, sealer
will apply the diff change of current Clusterfile and the original one.

```
sealer apply [flags]
```

### Examples

```
sealer apply -f Clusterfile
```

### Options

```
  -f, --Clusterfile string   Clusterfile path to apply a Kubernetes cluster (default "Clusterfile")
      --force                force to delete the specified cluster if set true
  -h, --help                 help for apply
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

