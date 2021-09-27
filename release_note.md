# release note

## v0.5.0

*Bugfix:*

- [x] Layer Not found appears when apply is executed on machines other than master0.
- [x] Lite build cache sectional images due to registry not start.
- [x] sealer login x509 err. (Set env ` SKIP_TLS_VERIFY=true` to skip )

*Optimize:*

- [x] Optimize the delete command.

   ```shell
   delete to cluster by baremetal provider:
       sealer delete --masters x.x.x.x --nodes x.x.x.x
       sealer delete --masters x.x.x.x-x.x.x.y --nodes x.x.x.x-x.x.x.y
   delete to cluster by cloud provider, just set the number of masters or nodes:
       sealer delete --masters 2 --nodes 3
   specify the cluster name(If there is only one cluster in the $HOME/.sealer directory, it should be applied. ):
       sealer delete --masters 2 --nodes 3 -f /root/.sealer/specify-cluster/Clusterfile
   delete all:
       sealer delete --all [--force]
       sealer delete -f /root/.sealer/mycluster/Clusterfile [--force]
   ``` 

- [x] Optimize the lite build step.
- [x] Shell plugin support on field.
- [x] Show more image details (creation time and size).