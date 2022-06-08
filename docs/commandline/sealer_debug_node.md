## sealer debug node

Debug node

```
sealer debug node [flags]
```

### Options

```
  -h, --help   help for node
```

### Options inherited from parent commands

```
      --check-list strings         Check items, such as network, volume.
      --config string              config file of sealer tool (default is $HOME/.sealer.json)
  -d, --debug                      turn on debug mode
  -e, --env stringToString         Environment variables to set in the container. (default [])
      --hide-path                  hide the log path
      --hide-time                  hide the log time
      --image string               Container image to use for debug container.
      --image-pull-policy string   Container image pull policy, default policy is IfNotPresent. (default "IfNotPresent")
      --name string                Container name to use for debug container.
  -n, --namespace string           Namespace of Pod. (default "default")
  -i, --stdin                      Keep stdin open on the container, even if nothing is attached.
  -t, --tty                        Allocate a TTY for the debugging container.
```

### SEE ALSO

* [sealer debug](sealer_debug.md)	 - Create debugging sessions for pods and nodes

