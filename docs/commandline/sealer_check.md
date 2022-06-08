## sealer check

check the state of cluster

### Synopsis

check command is used to check status of the cluster, including node status
, service status and pod status.

```
sealer check [flags]
```

### Examples

```
sealer check --pre or sealer check --post
```

### Options

```
  -h, --help   help for check
      --post   Check the status of the cluster after it is created
      --pre    Check dependencies before cluster creation
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

