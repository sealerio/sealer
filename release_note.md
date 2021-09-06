## release note

### v0.4.0

- [x] Fixed lite Build not cache images in manifest/*.yaml
- [x] Cache docker image for build
- [x] Add etcd, shell, label, hostname plugin config feature
  https://github.com/alibaba/sealer/blob/main/docs/design/plugin.md
- [x] Fixed not requiring login except for Cloud build
- [x] Fixed build to support submitting private registry account passwords
- [x] Fixed exceptions when executing kubectl or helm commands during lite builds