## sealer save

save image

### Synopsis

save image to a file 

```
sealer save [flags]
```

### Examples

```

sealer save -o [output file name] [image name]
save kubernetes:v1.18.3 image to kubernetes.tar.gz file:
sealer save -o kubernetes.tar.gz kubernetes:v1.18.3
```

### Options

```
  -h, --help            help for save
  -o, --output string   write the image to a file
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

