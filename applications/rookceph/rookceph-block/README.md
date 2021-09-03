# Overview

This image will create CephBlockPool with three replicas and also will create default storage class
named `rook-ceph-block` for use.

Components included in this image:

* 1 Deployment for rookceph operator.
* 3 ceph mons for ceph cluster.
* 1 ceph mgr for ceph cluster.
* enable ceph dashboard with ssl port 8443.

# How to run it

Use default Clusterfile to apply the ceph cluster.

see : [default ceph block Clusterfile examples](/applications/rookceph/rookceph-block/examples/Clusterfile.yaml)

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

Use ceph as the block storage backend to deploy mysql application.

see : [mysql with ceph block examples](/applications/rookceph/rookceph-block/examples/examples.yaml)

# How to rebuild it

Modify manifest.yaml or cephblockpool.yaml file according to your needs, then run below command to rebuild it.

```shell
sealer build -t {Your Image Name} -f Kubefile -b cloud .
```

More parameters see : https://rook.io/docs/rook/v1.7/ceph-block.html