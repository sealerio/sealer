## sealer exec

exec a shell command or script on specified nodes.

```
sealer exec [flags]
```

### Examples

```

exec to default cluster: my-cluster
	sealer exec "cat /etc/hosts"
specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
    sealer exec -c my-cluster "cat /etc/hosts"
set role label to exec cmd:
    sealer exec -c my-cluster -r master,slave,node1 "cat /etc/hosts"		

```

### Options

```
  -c, --cluster-name string   specify the name of cluster
  -h, --help                  help for exec
  -r, --roles string          set role label to roles
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

