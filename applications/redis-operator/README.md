# redis operator

Make sure you have the default storage class:

```
[root@iZ2zeaxzynknewlf0dkin9Z redis]# kubectl get sc
NAME                 PROVISIONER        RECLAIMPOLICY   VOLUMEBINDINGMODE      ALLOWVOLUMEEXPANSION   AGE
hostpath (default)   openebs.io/local   Delete          WaitForFirstConsumer   false                  25h
openebs-device       openebs.io/local   Delete          WaitForFirstConsumer   false                  25h
openebs-hostpath     openebs.io/local   Delete          WaitForFirstConsumer   false                  25h
```

Use the redis:

```
docker run -it --rm redis redis-cli -h rfs-redisfailover -p 26379
```

If you want to use it out of cluster, you should change the svc type to NodePort.
