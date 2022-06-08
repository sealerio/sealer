## sealer build

build a ClusterImage from a Kubefile

### Synopsis

build command is used to build a ClusterImage from specified Kubefile.
It organizes the specified Kubefile and input building context, and builds
a brand new ClusterImage.

```
sealer build [flags] PATH
```

### Examples

```
the current path is the context path, default build type is lite and use build cache

build:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 .

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --no-cache .

build without base:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --base=false .

build with args:
	sealer build -f Kubefile -t my-kubernetes:1.19.8 --build-arg MY_ARG=abc,PASSWORD=Sealer123 .

```

### Options

```
      --base                build with base image, default value is true. (default true)
      --build-arg strings   set custom build args
  -h, --help                help for build
  -t, --imageName string    the name of ClusterImage
  -f, --kubefile string     Kubefile filepath (default "Kubefile")
  -m, --mode string         ClusterImage build type, default is lite (default "lite")
      --no-cache            build without cache
      --platform string     set ClusterImage platform. If not set, keep same platform with runtime
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

