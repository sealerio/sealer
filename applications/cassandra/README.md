# USAGE

Cassandra can be accessed through the following URLs from within the cluster:

- CQL: my-cassandra.cassandra-system.svc.cluster.local:9042
- Thrift: my-cassandra.cassandra-system.svc.cluster.local:9160

To get your password run:

export CASSANDRA_PASSWORD=$(kubectl get secret --namespace "cassandra-system" my-cassandra -o jsonpath="{.data.cassandra-password}" | base64 --decode)

Check the cluster status by running:

kubectl exec -it --namespace cassandra-system $(kubectl get pods --namespace cassandra-system -l app=cassandra,release=my-cassandra -o jsonpath='{.items[0].metadata.name}') nodetool status

To connect to your Cassandra cluster using CQL:

1. Run a Cassandra pod that you can use as a client:

   kubectl run --namespace cassandra-system my-cassandra-client --rm --tty -i --restart='Never' \
   --env CASSANDRA_PASSWORD=$CASSANDRA_PASSWORD \
   \
   --image docker.io/bitnami/cassandra:4.0.0-debian-10-r0 -- bash

2. Connect using the cqlsh client:

   cqlsh -u cassandra -p $CASSANDRA_PASSWORD my-cassandra

To connect to your database from outside the cluster execute the following commands:

kubectl port-forward --namespace cassandra-system svc/my-cassandra 9042:9042 &
cqlsh -u cassandra -p $CASSANDRA_PASSWORD 127.0.0.1 9042

