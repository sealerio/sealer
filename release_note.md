# release note

## v0.5.1

*Bugfix:*

- [x] fix null point error caused by using label plugin.
- [x] fix imageList containing spaces causing panic.

*Optimize:*

- [x] lite build support add 'yml' file and skip to run cmd when its layer value contains "kubectl".
- [x] local build support cache image from imageList，manifests/*.yaml，charts.
- [x] support overwrite kubeadm config.