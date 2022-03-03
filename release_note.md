# release note

## v0.7.0

*✅ bugfix*:

- [x] [fixed sealer merged images losing layers](https://github.com/alibaba/sealer/issues/1019)
- [x] [fix building the wrong rootfs mount directory](https://github.com/alibaba/sealer/issues/1029)
- [x] [fixed scaling down node not completely cleaning up the environment](https://github.com/alibaba/sealer/issues/1042)
- [x] [fix `$rootfs/etc/kubeadm.yml' only partial kubeadm configuration causes failure](https://github.com/alibaba/sealer/issues/1046)
- [x] [fix failing to run sealer check --pre](https://github.com/alibaba/sealer/issues/1047)
- [x] [fix sealer exec does not show cmd result](https://github.com/alibaba/sealer/issues/1048)
- [x] [fix execute the join command repeatedly failed](https://github.com/alibaba/sealer/issues/1065)

*🚀 feat*:

- [x] add ipvs router if multi network interface available
- [x] sealer build support copy wildcards
- [x] [trigger download image based on directory name](https://github.com/alibaba/sealer/pull/1061)
- [x] [config support deep merge](https://github.com/alibaba/sealer/issues/1038)
- [x] [show CloudImage OS/ARCH info in docker hub](https://github.com/alibaba/sealer/issues/1022)
- [x] sealer delete node needs to be stable enough to be as error-free as possible
- [x] [optimize shell plugin](https://github.com/alibaba/sealer/issues/1052)
- [x] [env render in CMD](https://github.com/alibaba/sealer/issues/1083)
- [x] [create config file if file not exist](https://github.com/alibaba/sealer/issues/1085)
- [x] [!! rootfs plugin dir rename to plugins](https://github.com/alibaba/sealer/issues/1094)
- [x] [build kubefile support parser continuation character](https://github.com/alibaba/sealer/pull/1100)
- [x] [build support copy remote context](https://github.com/alibaba/sealer/issues/1130)
- [x] [generate Clusterfile to takeover a cluster](https://github.com/alibaba/sealer/pull/1171)
- [x] [add sans to cert after cluster already installed](https://github.com/alibaba/sealer/pull/1158)
- [x] sealer run support ip range
- [x] [support custom registry domain port](https://github.com/alibaba/sealer/pull/1152)
- [x] [set cmd arg for sealer run](https://github.com/alibaba/sealer/pull/1132)
- [x] [build support copy local docker image to cloud image](https://github.com/alibaba/sealer/issues/1133)
- [x] [run support custom port number](https://github.com/alibaba/sealer/pull/1082)