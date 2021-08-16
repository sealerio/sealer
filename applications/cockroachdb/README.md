# USAGE
CockroachDB can be accessed via port 26257 at the
following DNS name from within your cluster:

my-cockroachdb-public.cockroachdb-system.svc.cluster.local

Because CockroachDB supports the PostgreSQL wire protocol, you can connect to
the cluster using any available PostgreSQL client.

For example, you can open up a SQL shell to the cluster by running:

    kubectl run -it --rm cockroach-client \
        --image=cockroachdb/cockroach \
        --restart=Never \
        --command -- \
        ./cockroach sql --insecure --host=my-cockroachdb-public.cockroachdb-system

From there, you can interact with the SQL shell as you would any other SQL
shell, confident that any data you write will be safe and available even if
parts of your cluster fail.

Finally, to open up the CockroachDB admin UI, you can port-forward from your
local machine into one of the instances in the cluster:

    kubectl port-forward my-cockroachdb-0 8080

Then you can access the admin UI at http://localhost:8080/ in your web browser.

For more information on using CockroachDB, please see the project's docs at:
https://www.cockroachlabs.com/docs/
