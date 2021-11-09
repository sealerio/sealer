## sealer build

cloud image local build command line

### Synopsis

sealer build -f Kubefile -t my-kubernetes:1.19.9 [--mode cloud|container|lite] [--no-cache]

```
sealer build [flags] PATH
```

### Examples

```
the current path is the context path ,default build type is cloud and use build cache

cloud build :
	sealer build -f Kubefile -t my-kubernetes:1.19.9

container build :
	sealer build -f Kubefile -t my-kubernetes:1.19.9 -m container

lite build:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --mode lite

build without cache:
	sealer build -f Kubefile -t my-kubernetes:1.19.9 --no-cache

```

### Options

```
  -m, --mode string   cluster image build type,default is cloud
  -h, --help               help for build
  -t, --imageName string   cluster image name
  -f, --kubefile string    kubefile filepath (default "Kubefile")
      --no-cache           build without cache
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

