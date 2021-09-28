## sealer debug

Creating debugging sessions for pods and nodes

### Options

```
      --check-list strings         Check items, such as network、volume.
  -e, --env stringToString         Environment variables to set in the container. (default [])
  -h, --help                       help for debug
      --image string               Container image to use for debug container.
      --image-pull-policy string   Container image pull policy, default policy is IfNotPresent. (default "IfNotPresent")
      --name string                Container name to use for debug container.
  -n, --namespace string           Namespace of Pod. (default "default")
  -i, --stdin                      Keep stdin open on the container, even if nothing is attached.
  -t, --tty                        Allocate a TTY for the debugging container.
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 
* [sealer debug clean](sealer_debug_clean.md)	 - Clean the debug container od pod
* [sealer debug node](sealer_debug_node.md)	 - Debug node
* [sealer debug pod](sealer_debug_pod.md)	 - Debug pod or container
* [sealer debug show-images](sealer_debug_show-images.md)	 - List default images

