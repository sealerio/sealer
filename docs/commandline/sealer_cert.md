## sealer cert

update Kubernetes API server's cert

### Synopsis

Add domain or ip in certs:
    you had better backup old certs first.
	sealer cert --alt-names sealer.cool,10.103.97.2,127.0.0.1,localhost
    using "openssl x509 -noout -text -in apiserver.crt" to check the cert
	will update cluster API server cert, you need to restart your API server manually after using sealer cert.

    For example: add an EIP to cert.
    1. sealer cert --alt-names 39.105.169.253
    2. update the kubeconfig, cp /etc/kubernetes/admin.conf .kube/config
    3. edit .kube/config, set the apiserver address as 39.105.169.253, (don't forget to open the security group port for 6443, if you using public cloud)
    4. kubectl get pod, to check if it works or not


```
sealer cert [flags]
```

### Options

```
      --alt-names string   add domain or ip in certs, sealer.cool or 10.103.97.2
  -h, --help               help for cert
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

