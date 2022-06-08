## sealer upgrade

upgrade specified Kubernetes cluster

### Synopsis

sealer upgrade imagename --cluster clustername

```
sealer upgrade [flags]
```

### Examples

```
sealer upgrade kubernetes:v1.19.9 --cluster my-cluster
```

### Options

```
  -c, --cluster string   the name of cluster
  -h, --help             help for upgrade
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

