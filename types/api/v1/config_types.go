/*
Copyright 2021 Alibaba Group.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Application config file:

Clusterfile:

apiVersion: sealer.io/v1
kind: Cluster
metadata:
  name: my-cluster
spec:
  image: registry.cn-qingdao.aliyuncs.com/sealer-app/my-SAAS-all-inone:latest
  provider: BAREMETAL
---
apiVersion: sealer.io/v1
kind: Config
metadata:
  name: mysql-config
spec:
  path: etc/mysql-config.yaml
  data: |
       mysql-user: root
       mysql-passwd: xxx
...
---
apiVersion: sealer.io/v1
kind: Config
metadata:
  name: redis-config
spec:
  path: etc/redis-config.yaml
  data: |
       redis-user: root
       redis-passwd: xxx
...

When apply this Clusterfile, sealer will generate some values file for application config. Named etc/mysql-config.yaml etc/redis-config.yaml.

So if you want to use this config, Kubefile is like this:

FROM kubernetes:v1.19.9
CMD helm install mysql -f etc/mysql-config.yaml
CMD helm install mysql -f etc/redis-config.yaml
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigSpec defines the desired state of Config
type ConfigSpec struct {
	// Enumeration value is "merge" and "overwrite". default value is "overwrite".
	// Only yaml files format are supported if strategy is "merge", this will deeply merge each yaml file section.
	// Otherwise, will overwrite the whole file content with config data.
	Strategy string `json:"strategy,omitempty"`
	// preprocess with processor: value|toJson|toBase64|toSecret
	Process string `json:"process,omitempty"`
	//Data config real data
	Data string `json:"data,omitempty"`
	// which app to use this config.
	APPName string `json:"appName,omitempty"`
	// where the Data write on,
	// it could be the relative path of rootfs (manifests/mysql.yaml).
	// Or work with APPName to form a complete relative path(application/apps/{APPName}/mysql.yaml)
	Path string `json:"path,omitempty"`
}

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Config is the Schema for the configs API
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
