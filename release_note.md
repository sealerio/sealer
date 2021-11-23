# release note

## v0.5.2

*Optimize:*

- [x] Sealer Build replaces -b with the -m parameter (mode);
- [x] Plugin Adds the type field to specify the plugin type.([plugin docs](https://github.com/alibaba/sealer/blob/main/docs/design/plugin.md));
- [x] optimize returns an error when build fails to pull the image from the imageList;
- [x] optimize the delete command;
- [x] optimize unnecessary warn logs.

*Feature:*

- [x] sealer support upgrade cluster;
- [x] sealer run support specific infra provider;
- [x] support rmi with partial id.