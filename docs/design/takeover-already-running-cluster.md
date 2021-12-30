# Takeover already existing cluster

## Motivations

If you already have an existing kubernetes cluster, and you want to use sealer to manage the cluster lifecycle, like join masters, scale up or down nodes, clean the cluster.

## Proposal

1. Use `gen` command to specify the nodes ssh password and kubeconfig file.
2. Query Cluster nodes info, get the requirement argument sealer needs.
3. Generate a Clusterfile for the cluster.
4. Use sealer to manage the cluster.

## Use cases

### Takeover a cluster

```shell script
sealer gen --passwd xxxx --kubeconfig ~/.kube/config --image kubernetes:v1.19.8
```

The takeover actually is to generate a Clusterfile by kubeconfig.
Sealer will call kubernetes API to get masters and nodes IP info, then generate a Clusterfile.

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