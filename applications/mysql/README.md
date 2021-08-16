#USAGE
Tip:

Watch the deployment status using the command: kubectl get pods -w --namespace mysql-system

Services:

echo Primary: my-mysql-primary.mysql-system.svc.cluster.local:3306
echo Secondary: my-mysql-secondary.mysql-system.svc.cluster.local:3306

Administrator credentials:

echo Username: root
echo Password : $(kubectl get secret --namespace mysql-system my-mysql -o jsonpath="{.data.mysql-root-password}" | base64 --decode)

To connect to your database:

1. Run a pod that you can use as a client:

   kubectl run my-mysql-client --rm --tty -i --restart='Never' --image  docker.io/bitnami/mysql:8.0.26-debian-10-r10 --namespace mysql-system --command -- bash

2. To connect to primary service (read/write):

   mysql -h my-mysql-primary.mysql-system.svc.cluster.local -uroot -p my_database

3. To connect to secondary service (read-only):

   mysql -h my-mysql-secondary.mysql-system.svc.cluster.local -uroot -p my_database
