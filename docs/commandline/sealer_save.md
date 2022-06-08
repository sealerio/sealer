## sealer save

save ClusterImage to a tar file

### Synopsis

sealer save -o [output file name] [image name]

```
sealer save [flags]
```

### Examples

```

save kubernetes:v1.19.8 image to kubernetes.tar file:

sealer save -o kubernetes.tar kubernetes:v1.19.8
```

### Options

```
  -h, --help              help for save
  -o, --output string     write the image to a file
      --platform string   set ClusterImage platform
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

