## sealer merge

merge multiple images into one

### Synopsis

sealer merge image1:latest image2:latest image3:latest ......

```
sealer merge [flags]
```

### Examples

```

merge images:
	sealer merge kubernetes:v1.19.9 mysql:5.7.0 redis:6.0.0 -t new:0.1.0

```

### Options

```
  -h, --help                  help for merge
      --platform string       set ClusterImage platform, if not set,keep same platform with runtime
  -t, --target-image string   target image name
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

