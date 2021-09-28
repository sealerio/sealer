## sealer debug show-images

List default images

```
sealer debug show-images [flags]
```

### Options

```
  -h, --help   help for show-images
```

### Options inherited from parent commands

```
      --check-list strings         Check items, such as network、volume.
      --config string              config file (default is $HOME/.sealer.json)
  -d, --debug                      turn on debug mode
  -e, --env stringToString         Environment variables to set in the container. (default [])
      --image string               Container image to use for debug container.
      --image-pull-policy string   Container image pull policy, default policy is IfNotPresent. (default "IfNotPresent")
      --name string                Container name to use for debug container.
  -n, --namespace string           Namespace of Pod. (default "default")
  -i, --stdin                      Keep stdin open on the container, even if nothing is attached.
  -t, --tty                        Allocate a TTY for the debugging container.
```

### SEE ALSO

* [sealer debug](sealer_debug.md)	 - Creating debugging sessions for pods and nodes

