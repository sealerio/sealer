## sealer completion

generate autocompletion script for bash

### Synopsis

Generate the autocompletion script for sealer for the bash shell.
To load completions in your current shell session:

	source <(sealer completion bash)

To load completions for every new session, execute once:

- Linux :
	## If bash-completion is not installed on Linux, please install the 'bash-completion' package
		sealer completion bash > /etc/bash_completion.d/sealer
	

```
sealer completion
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --config string   config file (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
```

### SEE ALSO

* [sealer](sealer.md)	 - 

