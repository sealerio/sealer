## sealer search

search ClusterImage in default registry

```
sealer search [flags]
```

### Examples

```
sealer search <imageDomain>/<imageRepo>/<imageName> ...
## default imageDomain: 'registry.cn-qingdao.aliyuncs.com', default imageRepo: 'sealer-io'
ex.:
  sealer search kubernetes seadent/rootfs docker.io/library/hello-world

```

### Options

```
  -h, --help   help for search
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

