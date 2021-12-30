# release note

## v0.6.0

### **Major upgrade!!!:**

1. [Cluster v2 design](https://github.com/alibaba/sealer/blob/main/docs/design/clusterfile-v2.md#clusterfile-v2-design):
    1. Delete provider field
    2. Add env field
    3. Modify hosts field, add ssh and env rewrite (Different node has different ssh config like passwd)

2. [Normal docker base image, integration kyverno](https://github.com/alibaba/sealer/issues/859)

3. [Support configuring Docker service to trust Sealer Docker Registry Service](https://github.com/alibaba/sealer/issues/852)

4. [Registry settings, like registry username passwd](https://github.com/alibaba/sealer/issues/856)

5. [Support build ARM CloudImage in AMD environment](https://github.com/alibaba/sealer/issues/857)

6. **[Support custom kubeadm configuration](https://github.com/alibaba/sealer/blob/main/docs/design/clusterfile-v2.md#using-kubeconfig-to-overwrite-kubeadm-configs)**

7. [Sealer ApplicationsImage & incremental updating](https://github.com/alibaba/sealer/pull/817)

8. [Sealer exec command (execute command at role related node)](https://github.com/alibaba/sealer/issues/952)

9. *Optimized the log. Add -d or --debug to the command to view complete information about the execution process*

10. [Support plugin out of tree](https://github.com/alibaba/sealer/blob/main/docs/site/src/docs/getting-started/plugin.md#out-of-tree-plugin)

11. [Added shell_plugin execution phase `Post_Clean`(custom cleanup script executed after cluster deletion)](https://github.com/alibaba/sealer/issues/917)

12. [Fix lite mode build cache docker image failure](https://github.com/alibaba/sealer/issues/898)