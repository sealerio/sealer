# Overview

OpenEBS provides block volume support through the iSCSI protocol. Install iSCSI tools and make sure that iSCSI service
is running. See [iSCSI installation](https://openebs.io/docs/user-guides/prerequisites)

There are three OpenEBS Data Engine included in this section:

* [cStor](https://openebs.io/docs/concepts/cstor)
* [Jiva](https://openebs.io/docs/concepts/jiva)
* [LocalPV](https://openebs.io/docs/concepts/localpv)

## About cStor

cStor provides enterprise grade features such as synchronous data replication, snapshots, clones, thin provisioning of
data, high resiliency of data, data consistency and on-demand increase of capacity or performance.

Components included in this image:

* StatefulSet resource for openebs-cstor-csi-controller.
* Deployment resource for openebs-ndm-operator.
* Deployment resource for cspc-operator.
* Deployment resource for cvc-operator.
* Deployment resource for openebs-cstor-admission-server.
* DaemonSet resource for openebs-cstor-csi-node .
* DaemonSet resource for openebs-ndm.

## How to run it

1, Apply a kubernetes cluster with "openebs-cstor" installed.

use example/Clusterfile.yaml to apply "openebs-cstor" by modifying the image filed
as `image: registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-cstor:2.11.0`,and then run
command `sealer apply -f Clusterfile.yaml`

2, Create [CStorPoolCluster](https://openebs.io/docs/user-guides/cstor) to make
the storage cluster in ready status. example see cstor/cspc.yaml

## How to use it

1, use it as base cluster to deploy StatefulSet application.

After perform all the installation steps,you need get the default StorageClass name by run command:`kubectl get sc`.

To deploy a sample application using the default StorageClass: "cstor-csi-disk"

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: cstor-pvc
spec:
  storageClassName: cstor-csi-disk
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
---
apiVersion: v1
kind: Pod
metadata:
  name: busybox
  namespace: default
spec:
  containers:
    - command:
        - sh
        - -c
        - 'date >> /mnt/openebs-csi/date.txt; hostname >> /mnt/openebs-csi/hostname.txt; sync; sleep 5; sync; tail -f /dev/null;'
      image: busybox
      imagePullPolicy: Always
      name: busybox
      volumeMounts:
        - mountPath: /mnt/openebs-csi
          name: demo-vol
  volumes:
    - name: demo-vol
      persistentVolumeClaim:
        claimName: cstor-pvc
```

2, use it as base image to build anther cloud image

Kubefile:

```shell
FROM registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-cstor:2.11.0
# add helm repo and run helm install
RUN helm repo add bitnami https://charts.bitnami.com/bitnami
CMD helm install my-kafka --create-namespace --namespace kafka-system --set global.storageClass=cstor-csi-disk bitnami/kafka --version 14.0.5
```

## About Jiva

Jiva is a lightweight storage engine that is recommended to use for low capacity workloads. The snapshot and storage
management features of the other cStor engine are more advanced and is recommended when snapshots are a need.

Components included in this image:

* StatefulSet resource for openebs-jiva-csi-controller.
* Deployment resource for openebs-localpv-provisioner.
* Deployment resource for jiva-operator.
* DaemonSet resource for openebs-jiva-csi-node.

## How to run it

Apply a kubernetes cluster with "openebs-jiva" installed. use example/Clusterfile.yaml to apply "openebs-jiva" by
modifying the image filed as `image: registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-jiva:2.11.0`,and then run
command `sealer apply -f Clusterfile.yaml`

## How use it

Same with cStor engine.

## About LocalPV

OpenEBS provides Dynamic PV provisioners for Kubernetes Local Volumes. A local volume implies that storage is available
only from a single node. A local volume represents a mounted local storage device such as a disk, partition or
directory.

Components included in this image:

* Deployment resource for openebs-localpv-provisioner.

## How to run it

Apply a kubernetes cluster with "openebs-localpv" installed. use example/Clusterfile.yaml to apply "openebs-localpv" by
modifying the image filed as `image: registry.cn-qingdao.aliyuncs.com/sealer-apps/openebs-localpv:2.11.0`,and then run
command `sealer apply -f Clusterfile.yaml`

## How to use it

Same with cStor engine.

## How to rebuild it

Modify manifest yaml file according to your needs, then run below command to rebuild it.

```shell
sealer build -t {Your Image Name} -f Kubefile -m cloud .
```

More information see [official document here](https://openebs.io).