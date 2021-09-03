# Overview

This image will create a CephObjectStore that starts the RGW service in the cluster with an S3 API and also will create
default storage class named `rook-ceph-bucket` for use.

Components included in this image:

* 1 Deployment for rookceph operator.
* 3 ceph mons for ceph cluster.
* 1 ceph mgr for ceph cluster.
* enable ceph dashboard with ssl port 8443.

# How to run it

Use default Clusterfile to apply the ceph cluster.

see : [default ceph object store Clusterfile examples](/applications/rookceph/rookceph-object/examples/Clusterfile.yaml)

# How to use it

Connect to ceph cluster using below tools.Then run `ceph status` to check the status of ceph cluster.

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rook-ceph-tools
  namespace: rook-ceph
  labels:
    app: rook-ceph-tools
spec:
  replicas: 1
  selector:
    matchLabels:
      app: rook-ceph-tools
  template:
    metadata:
      labels:
        app: rook-ceph-tools
    spec:
      dnsPolicy: ClusterFirstWithHostNet
      containers:
        - name: rook-ceph-tools
          image: rook/ceph:v1.7.2
          command: [ "/tini" ]
          args: [ "-g", "--", "/usr/local/bin/toolbox.sh" ]
          imagePullPolicy: IfNotPresent
          env:
            - name: ROOK_CEPH_USERNAME
              valueFrom:
                secretKeyRef:
                  name: rook-ceph-mon
                  key: ceph-username
            - name: ROOK_CEPH_SECRET
              valueFrom:
                secretKeyRef:
                  name: rook-ceph-mon
                  key: ceph-secret
          volumeMounts:
            - mountPath: /etc/ceph
              name: ceph-config
            - name: mon-endpoint-volume
              mountPath: /etc/rook
      volumes:
        - name: mon-endpoint-volume
          configMap:
            name: rook-ceph-mon-endpoints
            items:
              - key: data
                path: mon-endpoints
        - name: ceph-config
          emptyDir: { }
      tolerations:
        - key: "node.kubernetes.io/unreachable"
          operator: "Exists"
          effect: "NoExecute"
          tolerationSeconds: 5

```

Use ceph as the object store backend act as AWS S3.

create a bucket.

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name: ceph-bucket
spec:
  generateBucketName: rookbucket
  storageClassName: rook-ceph-bucket
```

Client Connections

```shell
#config-map, secret, OBC will part of default if no specific name space mentioned
export AWS_HOST=$(kubectl -n default get cm ceph-bucket -o jsonpath='{.data.BUCKET_HOST}')
export AWS_ACCESS_KEY_ID=$(kubectl -n default get secret ceph-bucket -o jsonpath='{.data.AWS_ACCESS_KEY_ID}' | base64 --decode)
export AWS_SECRET_ACCESS_KEY=$(kubectl -n default get secret ceph-bucket -o jsonpath='{.data.AWS_SECRET_ACCESS_KEY}' | base64 --decode)
```

To test the CephObjectStore we will install the s3cmd tool into the toolbox pod.

`yum -y install s3cmd`

Test the CephObjectStore to upload a file.

```shell
echo "Hello Rook" > /tmp/rookObj 
s3cmd put /tmp/rookObj --no-ssl --host=${AWS_HOST} --host-bucket=s3://rookbucket
```

Test the CephObjectStore to download and verify the file from the bucket.

```shell
s3cmd get s3://rookbucket/rookObj /tmp/rookObj-download --no-ssl --host=${AWS_HOST} --host-bucket=s3://rookbucket
cat /tmp/rookObj-download
```

# How to rebuild it

Modify manifest.yaml or cephobject.yaml file according to your needs, then run below command to rebuild it.

```shell
sealer build -t {Your Image Name} -f Kubefile -b cloud .
```

More parameters see : https://rook.io/docs/rook/v1.7/ceph-object.html