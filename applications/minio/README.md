# USAGE
MinIO&reg; can be accessed via port 9000 on the following DNS name from within your cluster:

my-minio.minio-system.svc.cluster.local

To get your credentials run:

export ACCESS_KEY=$(kubectl get secret --namespace minio-system my-minio -o jsonpath="{.data.access-key}" | base64 --decode)
export SECRET_KEY=$(kubectl get secret --namespace minio-system my-minio -o jsonpath="{.data.secret-key}" | base64 --decode)

To connect to your MinIO&reg; server using a client:

- Run a MinIO&reg; Client pod and append the desired command (e.g. 'admin info'):

  kubectl run --namespace minio-system my-minio-client \
  --rm --tty -i --restart='Never' \
  --env MINIO_SERVER_ACCESS_KEY=$ACCESS_KEY \
  --env MINIO_SERVER_SECRET_KEY=$SECRET_KEY \
  --env MINIO_SERVER_HOST=my-minio \
  --image docker.io/bitnami/minio-client:2021.7.27-debian-10-r7 -- admin info minio

To access the MinIO&reg; web UI:

- Get the MinIO&reg; URL:

  echo "MinIO&reg; web URL: http://127.0.0.1:9000/minio"
  kubectl port-forward --namespace minio-system svc/my-minio 9000:9000
