## sealer

A tool to build, share and run any distributed applications.

### Synopsis

sealer is a tool to seal application's all dependencies and Kubernetes
into ClusterImage by Kubefile, distribute this application anywhere via ClusterImage, 
and run it within any cluster with Clusterfile in one command.


### Options

```
      --config string   config file of sealer tool (default is $HOME/.sealer.json)
  -d, --debug           turn on debug mode
  -h, --help            help for sealer
      --hide-path       hide the log path
      --hide-time       hide the log time
  -t, --toggle          Help message for toggle
```

### SEE ALSO

* [sealer apply](sealer_apply.md)	 - apply a Kubernetes cluster via specified Clusterfile
* [sealer build](sealer_build.md)	 - build a ClusterImage from a Kubefile
* [sealer cert](sealer_cert.md)	 - update Kubernetes API server's cert
* [sealer check](sealer_check.md)	 - check the state of cluster
* [sealer completion](sealer_completion.md)	 - generate autocompletion script for bash
* [sealer debug](sealer_debug.md)	 - Create debugging sessions for pods and nodes
* [sealer delete](sealer_delete.md)	 - delete an existing cluster
* [sealer exec](sealer_exec.md)	 - exec a shell command or script on specified nodes.
* [sealer gen](sealer_gen.md)	 - generate a Clusterfile to take over a normal cluster which is not deployed by sealer
* [sealer gen-doc](sealer_gen-doc.md)	 - generate document for sealer CLI with MarkDown format
* [sealer images](sealer_images.md)	 - list all ClusterImages on the local node
* [sealer inspect](sealer_inspect.md)	 - print the image information or Clusterfile
* [sealer join](sealer_join.md)	 - join new master or worker node to specified cluster
* [sealer load](sealer_load.md)	 - load a ClusterImage from a tar file
* [sealer login](sealer_login.md)	 - login image registry
* [sealer merge](sealer_merge.md)	 - merge multiple images into one
* [sealer prune](sealer_prune.md)	 - prune sealer data dir
* [sealer pull](sealer_pull.md)	 - pull ClusterImage from a registry to local
* [sealer push](sealer_push.md)	 - push ClusterImage to remote registry
* [sealer rmi](sealer_rmi.md)	 - remove local images by name
* [sealer run](sealer_run.md)	 - start to run a cluster from a ClusterImage
* [sealer save](sealer_save.md)	 - save ClusterImage to a tar file
* [sealer search](sealer_search.md)	 - search ClusterImage in default registry
* [sealer tag](sealer_tag.md)	 - create a new tag that refers to a local ClusterImage
* [sealer upgrade](sealer_upgrade.md)	 - upgrade specified Kubernetes cluster
* [sealer version](sealer_version.md)	 - show sealer and related versions

