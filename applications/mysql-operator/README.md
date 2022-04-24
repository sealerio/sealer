# Access the cluster

Get root passwd:

```
[root@iZ2zeaxzynknewlf0dkin9Z ~]# kubectl get secret cluster1-secrets -oyaml
apiVersion: v1
data:
...
  root: TGlaUXlqbDQzbXNnVnRVeFQ=
...
kind: Secret
```

```
echo 'TGlaUXlqbDQzbXNnVnRVeFQ='|base64 -d
```

Get mysql host:

```
[root@iZ2zeaxzynknewlf0dkin9Z ~]# kubectl get svc
NAME                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                                 AGE
cluster1-haproxy                  ClusterIP   10.98.98.110    <none>        3306/TCP,3309/TCP,33062/TCP,33060/TCP   53m
cluster1-haproxy-replicas         ClusterIP   10.97.32.44     <none>        3306/TCP                                53m
```

```
mysql -h 10.97.32.44 -p
Enter password:
Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MySQL connection id is 3424
Server version: 8.0.23-14.1 Percona XtraDB Cluster (GPL), Release rel14, Revision d3b9a1d, WSREP version 26.4.3

Copyright (c) 2000, 2018, Oracle, MariaDB Corporation Ab and others.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

MySQL [(none)]> show databases;
+--------------------+
| Database           |
+--------------------+
| information_schema |
| mysql              |
| performance_schema |
| sys                |
+--------------------+
4 rows in set (0.00 sec)
```
