# sealer Roadmap

## Improve the basic capabilities of the Sealer Image

* Multi-architecture/hybrid architecture support
* Easy-to-use image caching capabilities with no intrusion
* Provide better `Sealer Image` collaboration mechanisms
  * open the `Sealer Image` Repo project
  * contains most of the common open source middleware and kxs runtime(not provided by the Sealer community)

## Focus the core of distributed application definition

* Keep exploring reducing complexity of k8s-based distributed application definition
* Support the full life cycle management of application and decoupled from the kxs runtime
* Support more built-in application types
  * contains common application types including helm chart, kube resource, OAM, etc
  * provide extensibility, supports user-defined extended app types
* Support ClusterImage on more heterogeneous infrastucture:
  * general OS and arch
  * general existing Kubernetes

## Build stable and easy-to-use cluster delivery capabilities

* support for large-scale cluster life cycle management, such as 3K nodes
* support for managed existing clusters
* support more kxs runtime support, such as k8s, k0s, k3s, etc
* support IAAS diagnostic capabilities, provider cluster pre-installation check tools

## Sealer spec Definition Stabilization

* Kubefile stabilization
* Clusterfile stabilization
* ClusterImage spec stabilization
  * define Kubernetes part and application part separately
  * protocol stabilization among sealer binary, basefs and cluster instance

## Performance Improvement

* Improve `Sealer Image` distribution efficiency as possible
* Improves `sealer build` efficiency as possible through caching and other capabilities
* Improve cluster bootstrap efficiency as possible

