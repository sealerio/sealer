# Takeover already existing cluster

## Motivations

If you already have an existing kubernetes cluster, and you want to use sealer to manage the cluster lifecycle, like
join masters, scale up or down nodes, clean the cluster.

## Proposal

1. Use `sealer gen` command to specify the nodes ssh password and cloud image name.
2. Query Cluster nodes info, get the requirement argument sealer needs.
3. Generate a Clusterfile for the cluster.
4. Use sealer to manage the cluster.

## Use cases

### Takeover a cluster

```shell
Usage:
  sealer gen [flags]

Flags:
  -h, --help               help for gen
      --image string       Set taken over cloud image
      --name string        Set taken over cluster name (default "default")
      --passwd string      Set taken over ssh passwd
      --pk string          set server private key (default "/root/.ssh/id_rsa")
      --pk-passwd string   set server private key password
      --port uint16        set the sshd service port number for the server (default port: 22) (default 22)
```

Example:

```shell script
sealer gen --passwd xxxx --image kubernetes:v1.19.8
```

The takeover actually is to generate a Clusterfile by kubeconfig. Sealer will call kubernetes API to get masters and
nodes IP info, then generate a Clusterfile.

Also, sealer will pull a CloudImage which matches the kubernetes version.

Then you can use any sealer command to manage the cluster like:

> Upgrade cluster

```shell script
sealer upgrade --image kubernetes:v1.22.0
```

> Scale

```shell script
sealer join --node x.x.x.x
```

> Deploy a CloudImage into the cluster

```shell script
sealer run mysql-cluster:5.8
```