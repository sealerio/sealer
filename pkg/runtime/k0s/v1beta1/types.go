/*
Copyright 2022 k0s authors

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

package v1beta1

import (
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ClusterSpec defines the desired state of ClusterConfig
type ClusterSpec struct {
	API               *APISpec               `json:"api"`
	ControllerManager *ControllerManagerSpec `json:"controllerManager,omitempty"`
	Scheduler         *SchedulerSpec         `json:"scheduler,omitempty"`
	Storage           *StorageSpec           `json:"storage"`
	Network           *Network               `json:"network"`
	PodSecurityPolicy *PodSecurityPolicy     `json:"podSecurityPolicy"`
	WorkerProfiles    WorkerProfiles         `json:"workerProfiles,omitempty"`
	Telemetry         *ClusterTelemetry      `json:"telemetry"`
	Install           *InstallSpec           `json:"installConfig,omitempty"`
	Images            *ClusterImages         `json:"images"`
	Extensions        *ClusterExtensions     `json:"extensions,omitempty"`
	Konnectivity      *KonnectivitySpec      `json:"konnectivity,omitempty"`
}

// ClusterConfigStatus defines the observed state of ClusterConfig
type ClusterConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:validation:Optional
// +genclient
// +genclient:onlyVerbs=create,delete,list,get,watch,update
// +groupName=k0s.k0sproject.io

// ClusterConfig is the Schema for the clusterconfigs API
type ClusterConfig struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
	metav1.TypeMeta   `json:",omitempty,inline"`

	Spec   *ClusterSpec        `json:"spec,omitempty"`
	Status ClusterConfigStatus `json:"status,omitempty"`
}

// APISpec defines the settings for the K0s API
type APISpec struct {
	// Local address on which to bind an API
	Address string `json:"address"`

	// The loadbalancer address (for k0s controllers running behind a loadbalancer)
	ExternalAddress string `json:"externalAddress,omitempty"`
	// TunneledNetworkingMode indicates if we access to KAS through konnectivity tunnel
	TunneledNetworkingMode bool `json:"tunneledNetworkingMode"`
	// Map of key-values (strings) for any extra arguments to pass down to Kubernetes api-server process
	ExtraArgs map[string]string `json:"extraArgs,omitempty"`
	// Custom port for k0s-api server to listen on (default: 9443)
	K0sAPIPort int `json:"k0sApiPort,omitempty"`

	// Custom port for kube-api server to listen on (default: 6443)
	Port int `json:"port"`

	// List of additional addresses to push to API servers serving the certificate
	SANs []string `json:"sans"`
}

// ControllerManagerSpec defines the fields for the ControllerManager
type ControllerManagerSpec struct {
	// Map of key-values (strings) for any extra arguments you want to pass down to the Kubernetes controller manager process
	ExtraArgs map[string]string `json:"extraArgs,omitempty"`
}

// SchedulerSpec defines the fields for the Scheduler
type SchedulerSpec struct {
	// Map of key-values (strings) for any extra arguments you want to pass down to Kubernetes scheduler process
	ExtraArgs map[string]string `json:"extraArgs,omitempty"`
}

// StorageSpec defines the storage related config options
type StorageSpec struct {
	Etcd *EtcdConfig `json:"etcd"`
	Kine *KineConfig `json:"kine,omitempty"`

	// Type of the data store (valid values:etcd or kine)
	Type string `json:"type"`
}

// EtcdConfig defines etcd related config options
type EtcdConfig struct {
	// ExternalCluster defines external etcd cluster related config options
	ExternalCluster *ExternalCluster `json:"externalCluster"`

	// Node address used for etcd cluster peering
	PeerAddress string `json:"peerAddress"`
}

// ExternalCluster defines external etcd cluster related config options
type ExternalCluster struct {
	// Endpoints of external etcd cluster used to connect by k0s
	Endpoints []string `json:"endpoints"`

	// EtcdPrefix is a prefix to prepend to all resource paths in etcd
	EtcdPrefix string `json:"etcdPrefix"`

	// CaFile is the host path to a file with CA certificate
	CaFile string `json:"caFile"`

	// ClientCertFile is the host path to a file with TLS certificate for etcd client
	ClientCertFile string `json:"clientCertFile"`

	// ClientKeyFile is the host path to a file with TLS key for etcd client
	ClientKeyFile string `json:"clientKeyFile"`
}

// KineConfig defines the Kine related config options
type KineConfig struct {
	// kine datasource URL
	DataSource string `json:"dataSource"`
}

// Network defines the network related config options
type Network struct {
	Calico     *Calico     `json:"calico"`
	DualStack  DualStack   `json:"dualStack,omitempty"`
	KubeProxy  *KubeProxy  `json:"kubeProxy"`
	KubeRouter *KubeRouter `json:"kuberouter"`

	// Pod network CIDR to use in the cluster
	PodCIDR string `json:"podCIDR"`
	// Network provider (valid values: calico, kuberouter, or custom)
	Provider string `json:"provider"`
	// Network CIDR to use for cluster VIP services
	ServiceCIDR string `json:"serviceCIDR,omitempty"`
	// Cluster Domain
	ClusterDomain string `json:"clusterDomain,omitempty"`
}

// Calico defines the calico related config options
type Calico struct {
	// Enable wireguard-based encryption (default: false)
	EnableWireguard bool `json:"wireguard"`

	// The host path for Calicos flex-volume-driver(default: /usr/libexec/k0s/kubelet-plugins/volume/exec/nodeagent~uds)
	FlexVolumeDriverPath string `json:"flexVolumeDriverPath"`

	// Host's IP Auto-detection method for Calico (see https://docs.projectcalico.org/reference/node/configuration#ip-autodetection-methods)
	IPAutodetectionMethod string `json:"ipAutodetectionMethod,omitempty"`

	// Host's IPv6 Auto-detection method for Calico
	IPv6AutodetectionMethod string `json:"ipV6AutodetectionMethod,omitempty"`

	// MTU for overlay network (default: 0)
	MTU int `json:"mtu" yaml:"mtu"`

	// vxlan (default) or ipip
	Mode string `json:"mode"`

	// Overlay Type (Always, Never or CrossSubnet)
	Overlay string `json:"overlay" validate:"oneof=Always Never CrossSubnet" `

	// The UDP port for VXLAN (default: 4789)
	VxlanPort int `json:"vxlanPort"`

	// The virtual network ID for VXLAN (default: 4096)
	VxlanVNI int `json:"vxlanVNI"`

	// Windows Nodes (default: false)
	WithWindowsNodes bool `json:"withWindowsNodes"`
}

// DualStack defines network configuration for ipv4\ipv6 mixed cluster setup
type DualStack struct {
	Enabled         bool   `json:"enabled,omitempty"`
	IPv6PodCIDR     string `json:"IPv6podCIDR,omitempty"`
	IPv6ServiceCIDR string `json:"IPv6serviceCIDR,omitempty"`
}

// KubeProxy defines the configuration for kube-proxy
type KubeProxy struct {
	Disabled bool   `json:"disabled,omitempty"`
	Mode     string `json:"mode,omitempty"`
}

// KubeRouter defines the kube-router related config options
type KubeRouter struct {
	// Auto-detection of used MTU (default: true)
	AutoMTU bool `json:"autoMTU"`
	// Override MTU setting (autoMTU must be set to false)
	MTU int `json:"mtu"`
	// Comma-separated list of global peer addresses
	PeerRouterASNs string `json:"peerRouterASNs"`
	// Comma-separated list of global peer ASNs
	PeerRouterIPs string `json:"peerRouterIPs"`
}

// PodSecurityPolicy defines the config options for setting system level default PSP
type PodSecurityPolicy struct {
	// default PSP for the cluster (00-k0s-privileged/99-k0s-restricted)
	DefaultPolicy string `json:"defaultPolicy"`
}

// WorkerProfiles profiles collection
type WorkerProfiles []WorkerProfile

// WorkerProfile worker profile
type WorkerProfile struct {
	// String; name to use as profile selector for the worker process
	Name string `json:"name"`
	// Worker Mapping object
	Config json.RawMessage `json:"values"`
}

// ClusterTelemetry holds telemetry related settings
type ClusterTelemetry struct {
	Enabled bool `json:"enabled"`
}

// InstallSpec defines the required fields for the `k0s install` command
type InstallSpec struct {
	SystemUsers *SystemUser `json:"users,omitempty"`
}

// SystemUser defines the user to use for each component
type SystemUser struct {
	Etcd          string `json:"etcdUser,omitempty"`
	Kine          string `json:"kineUser,omitempty"`
	Konnectivity  string `json:"konnectivityUser,omitempty"`
	KubeAPIServer string `json:"kubeAPIserverUser,omitempty"`
	KubeScheduler string `json:"kubeSchedulerUser,omitempty"`
}

// ClusterImages sets docker images for addon components
type ClusterImages struct {
	Konnectivity  ImageSpec `json:"konnectivity"`
	PushGateway   ImageSpec `json:"pushgateway"`
	MetricsServer ImageSpec `json:"metricsserver"`
	KubeProxy     ImageSpec `json:"kubeproxy"`
	CoreDNS       ImageSpec `json:"coredns"`

	Calico     CalicoImageSpec     `json:"calico"`
	KubeRouter KubeRouterImageSpec `json:"kuberouter"`

	Repository        string `json:"repository,omitempty"`
	DefaultPullPolicy string `json:"default_pull_policy,omitempty"`
}

// ImageSpec container image settings
type ImageSpec struct {
	Image   string `json:"image"`
	Version string `json:"version"`
}

// CalicoImageSpec config group for calico related image settings
type CalicoImageSpec struct {
	CNI             ImageSpec `json:"cni"`
	Node            ImageSpec `json:"node"`
	KubeControllers ImageSpec `json:"kubecontrollers"`
}

// KubeRouterImageSpec config group for kube-router related images
type KubeRouterImageSpec struct {
	CNI          ImageSpec `json:"cni"`
	CNIInstaller ImageSpec `json:"cniInstaller"`
}

// ClusterExtensions specifies cluster extensions
type ClusterExtensions struct {
	Storage *StorageExtension `json:"storage"`
	Helm    *HelmExtensions   `json:"helm"`
}

// StorageExtension specifies cluster default storage
type StorageExtension struct {
	Type                      string `json:"type"`
	CreateDefaultStorageClass bool   `json:"create_default_storage_class"`
}

// HelmExtensions specifies settings for cluster helm based extensions
type HelmExtensions struct {
	Repositories RepositoriesSettings `json:"repositories"`
	Charts       ChartsSettings       `json:"charts"`
}

// RepositoriesSettings repository settings
type RepositoriesSettings []Repository

// Repository describes single repository entry. Fields map to the CLI flags for the "helm add" command
type Repository struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	CAFile   string `json:"caFile"`
	CertFile string `json:"certFile"`
	Insecure bool   `json:"insecure"`
	KeyFile  string `json:"keyfile"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChartsSettings charts settings
type ChartsSettings []Chart

// Chart single helm addon
type Chart struct {
	Name      string        `json:"name"`
	ChartName string        `json:"chartname"`
	Version   string        `json:"version"`
	Values    string        `json:"values"`
	TargetNS  string        `json:"namespace"`
	Timeout   time.Duration `json:"timeout"`
}

// KonnectivitySpec defines the requested state for Konnectivity
type KonnectivitySpec struct {
	// agent port to listen on (default 8132)
	AgentPort int64 `json:"agentPort,omitempty"`
	// admin port to listen on (default 8133)
	AdminPort int64 `json:"adminPort,omitempty"`
}
