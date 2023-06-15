/*
Copyright 2021 alibaba.

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

package v2

import (
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Foo is an example field of Cluster. Edit Cluster_types.go to remove/update
	Image string `json:"image,omitempty"`
	// Why env not using map[string]string
	// Because some argument is list, like: CertSANS=127.0.0.1 CertSANS=localhost, if ENV is map, will merge those two values
	// but user want to config a list, using array we can convert it to {CertSANS:[127.0.0.1, localhost]}
	Env     []string `json:"env,omitempty"`
	CMDArgs []string `json:"cmd_args,omitempty"`
	CMD     []string `json:"cmd,omitempty"`
	// APPNames This field allows user to specify the app name they want to run launch.
	APPNames         []string               `json:"appNames,omitempty"`
	Hosts            []Host                 `json:"hosts,omitempty"`
	SSH              v1.SSH                 `json:"ssh,omitempty"`
	ContainerRuntime ContainerRuntimeConfig `json:"containerRuntime,omitempty"`
	// HostAliases holds the mapping between IP and hostnames that will be injected as an entry in the
	// host's hosts file.
	HostAliases []HostAlias `json:"hostAliases,omitempty"`
	// Registry field contains configurations about local registry and remote registry.
	Registry Registry `json:"registry,omitempty"`

	// DataRoot set sealer rootfs directory path.
	// if not set, default value is "/var/lib/sealer/data"
	DataRoot string `json:"dataRoot,omitempty"`
}

type ContainerRuntimeConfig struct {
	Type string `json:"type,omitempty"`
}

type Host struct {
	IPS   []net.IP `json:"ips,omitempty"`
	Roles []string `json:"roles,omitempty"`
	//overwrite SSH config
	SSH v1.SSH `json:"ssh,omitempty"`
	//overwrite env
	Env    []string          `json:"env,omitempty"`
	Labels map[string]string `json:"labels,omitempty"`
	Taints []string          `json:"taints,omitempty"`
}

// HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the
// pod's hosts file.
type HostAlias struct {
	// IP address of the host file entry.
	IP string `json:"ip,omitempty"`
	// Hostnames for the above IP address.
	Hostnames []string `json:"hostnames,omitempty"`
}

type Registry struct {
	// LocalRegistry is the sealer builtin registry configuration
	LocalRegistry *LocalRegistry `json:"localRegistry,omitempty"`
	// ExternalRegistry used to serve external registry service. do not support yet.
	ExternalRegistry *ExternalRegistry `json:"externalRegistry,omitempty"`
}

type RegistryConfig struct {
	Domain   string `json:"domain,omitempty"`
	Port     int    `json:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

type ExternalRegistry struct {
	RegistryConfig
}

type LocalRegistry struct {
	RegistryConfig
	// HA indicate that whether local registry will be deployed on all master nodes.
	// if LocalRegistry is not specified, default value is true.
	HA *bool `json:"ha,omitempty"`
	// Insecure indicated that whether the local registry is exposed in HTTPS.
	// if true sealer will not generate default ssl cert.
	Insecure *bool   `json:"insecure,omitempty"`
	Cert     TLSCert `json:"cert,omitempty"`
}

type TLSCert struct {
	SubjectAltName *SubjectAltName `json:"subjectAltName,omitempty"`
}

type SubjectAltName struct {
	DNSNames []string `json:"dnsNames,omitempty"`
	IPs      []string `json:"ips,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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

func (in *Cluster) GetMasterIPList() []net.IP {
	return in.GetIPSByRole(common.MASTER)
}

func (in *Cluster) GetMasterIPStrList() (ipStrList []string) {
	ipList := in.GetIPSByRole(common.MASTER)

	for _, ip := range ipList {
		ipStrList = append(ipStrList, ip.String())
	}

	return ipStrList
}

func (in *Cluster) GetNodeIPList() []net.IP {
	return in.GetIPSByRole(common.NODE)
}

func (in *Cluster) GetAllIPList() []net.IP {
	return append(in.GetIPSByRole(common.MASTER), in.GetIPSByRole(common.NODE)...)
}

func (in *Cluster) GetMaster0IP() net.IP {
	masterIPList := in.GetIPSByRole(common.MASTER)
	if len(masterIPList) == 0 {
		return nil
	}
	return masterIPList[0]
}

func (in *Cluster) GetIPSByRole(role string) []net.IP {
	var hosts []net.IP
	for _, host := range in.Spec.Hosts {
		for _, hostRole := range host.Roles {
			if role == hostRole {
				hosts = append(hosts, host.IPS...)
				continue
			}
		}
	}
	return hosts
}

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
