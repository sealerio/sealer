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
	"time"

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
	APPNames []string `json:"appNames,omitempty"`
	Hosts    []Host   `json:"hosts,omitempty"`
	SSH      v1.SSH   `json:"ssh,omitempty"`
	// HostAliases holds the mapping between IP and hostnames that will be injected as an entry in the
	// host's hosts file.
	HostAliases []HostAlias `json:"hostAliases,omitempty"`
	// Registry field contains configurations about local registry and remote registry.
	Registry Registry `json:"registry,omitempty"`

	Apps []App `json:"apps,omitempty"`
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

type App struct {
	// the AppName
	AppName string `json:"appName,omitempty"`

	AppFiles []AppFiles `json:"appFiles,omitempty"`

	// app Launch customization
	Launch *Launch `json:"launch,omitempty"`

	// app Delete customization
	Delete *Delete `json:"delete,omitempty"`
}

type ValueType string

const (
	RawValueType     ValueType = "raw"
	SectionValueType ValueType = "section"
	ArgsValueType    ValueType = "args"
)

type PreProcessor string

const (
	ToSecretPreProcessor    PreProcessor = "toSecret"
	ToNamespacePreProcessor PreProcessor = "toNamespace"
)

type AppFiles struct {
	// FilePath represents the path to write the Values, required.
	FilePath string `json:"filePath,omitempty"`

	//PreProcessor pre mutate the whole Values. The premise is ValueType must be RawValueType.
	//ToSecretPreProcessor: mutate to kubernetes Secrete.
	//ToNamespacePreProcessor: mutate to kubernetes Namespace.
	PreProcessor string `json:"preProcessor,omitempty"`

	//ValueType support as blew:
	// RawValueType: this will overwrite the FilePath or work with PreProcessor to mutate the Values.
	// SectionValueType: Only yaml files format are supported, this type will deeply merge each yaml file section.
	// ArgsValueType: this will render the FilePath
	ValueType string `json:"valueType,omitempty"`

	// Values real app launch need.
	// it could be raw content, yaml data, yaml section data, key-value pairs, and so on.
	Values []byte `json:"values,omitempty"`
}

type Delete struct {
	// raw cmds support
	Cmds []string `json:"cmds,omitempty"`
}

type Launch struct {
	// Cmds raw cmds support, not required, exclusive with app type.
	Cmds []string `json:"cmds,omitempty"`

	// Helm represents the helm app type
	Helm *Helm `json:"helm,omitempty"`

	// Shell represents the shell app type
	Shell *Shell `json:"shell,omitempty"`

	// Kube represents the kube app type,
	// The reason why this is an arrays that it can support operations on resources in different namespaces.
	Kube []Kubectl `json:"kube,omitempty"`
}

type Helm struct {
	// ChartName will omit the chart values NAME parameter.
	ChartName string `json:"chartName,omitempty"`

	//CreateNamespace: create the release namespace if not present
	CreateNamespace bool `json:"createNamespace,omitempty"`

	//DisableHooks: prevent hooks from running during install
	DisableHooks bool `json:"disableHooks,omitempty"`

	//SkipCRDs: if set, no CRDs will be installed. By default, CRDs are installed if not already present
	SkipCRDs bool `json:"skipCRDs,omitempty"`

	//Timeout to wait for any individual Kubernetes operation (like Jobs for hooks)
	Timeout time.Duration `json:"timeout,omitempty"`

	// Wait: if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet
	// are in a ready state before marking the release as successful. It will wait for as long as Timeout
	Wait bool `json:"wait,omitempty"`

	// ValueFiles specify values in a YAML file or a URL, it can specify multiple.
	ValueFiles []string `json:"valueFiles,omitempty"`

	//set Values on the command line ,it can specify multiple or separate values with commas: key1=val1,key2=val2.
	Values []string `json:"values,omitempty"`
}

type Shell struct {
	// the environment variables to execute the shell file
	Envs []string `json:"envs,omitempty"`

	//FilePath represents the shell file path
	FilePaths []string `json:"filePaths,omitempty"`
}

type Kubectl struct {
	//FileNames represents the resources applied from
	FileNames []string `json:"fileNames,omitempty"`

	//Directory represents the resources applied from
	Directory string `json:"directory,omitempty"`

	// Namespace apply resources to specific namespace.
	Namespace string `json:"namespace,omitempty"`

	// Action represents kubectl command type,such as apply or create.
	Action string `json:"action,omitempty"`
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
