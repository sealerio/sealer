// Copyright © 2021 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1

import (
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SSH struct {
	Encrypted bool   `json:"encrypted,omitempty"`
	User      string `json:"user,omitempty"`
	Passwd    string `json:"passwd,omitempty"`
	Pk        string `json:"pk,omitempty"`
	PkPasswd  string `json:"pkPasswd,omitempty"`
	Port      string `json:"port,omitempty"`
}

type Network struct {
	PodCIDR string `json:"podCIDR,omitempty"`
	SvcCIDR string `json:"svcCIDR,omitempty"`
}

type Hosts struct {
	CPU        string   `json:"cpu,omitempty"`
	Memory     string   `json:"memory,omitempty"`
	Count      string   `json:"count,omitempty"`
	SystemDisk string   `json:"systemDisk,omitempty"`
	DataDisks  []string `json:"dataDisks,omitempty"`
	IPList     []net.IP `json:"ipList,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Cluster. Edit Cluster_types.go to remove/update
	Image    string   `json:"image,omitempty"`
	Env      []string `json:"env,omitempty"`
	Provider string   `json:"provider,omitempty"`
	SSH      SSH      `json:"ssh,omitempty"`
	Network  Network  `json:"network,omitempty"`
	CertSANS []string `json:"certSANS,omitempty"`
	Masters  Hosts    `json:"masters,omitempty"`
	Nodes    Hosts    `json:"nodes,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// TODO save cluster status info

}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

//func (cluster *Cluster) GetClusterEIP() string {
//	return cluster.Annotations[common.Eip]
//}
//
//func (cluster *Cluster) GetClusterMaster0IP() string {
//	return cluster.Annotations[common.Master0InternalIP]
//}
//
//func (cluster *Cluster) GetEipID() string {
//	return cluster.Annotations[common.EipID]
//}
//
//func (cluster *Cluster) GetMaster0ID() string {
//	return cluster.Annotations[common.Master0ID]
//}
//
//func (cluster *Cluster) GetVpcID() string {
//	return cluster.Annotations[common.VpcID]
//}
//
//func (cluster *Cluster) GetVSwitchID() string {
//	return cluster.Annotations[common.VSwitchID]
//}

func (in *Cluster) GetAnnotationsByKey(key string) string {
	return in.Annotations[key]
}

func (in *Cluster) SetAnnotations(key, value string) {
	if in.Annotations == nil {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[key] = value
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
