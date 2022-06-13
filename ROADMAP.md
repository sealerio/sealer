# sealer Roadmap

## Focus the core of distributed application definition

* Keep exploring reducing complexity of k8s-based distributed application definition
* Support ClusterImage on more heterogeneous infrastucture:
  * general OS and arch
  * general existing Kubernetes

## sealer spec Definition Stabilization

* Kubefile stabilization
* Clusterfile stabilization
* ClusterImage spec stabilization
  * define Kubernetes part and application part separately
  * protocol stabilization among sealer binary, basefs and cluster instance

## Neutral Architecture and Cloud Native Ecosystem Integration

* Modularize architecture, including:
  * data structure
  * control workflow
  * data flow
* HA built-in image registry
* others

## Performance Improvement

* Improve cluster bootstrap efficiency as possible
* Improve ClusterImage distribution efficiency as possible
