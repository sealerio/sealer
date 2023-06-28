/*
Copyright 2023 alibaba.

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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	//Cmds raw command line which has the highest priority, is mutually exclusive with the AppNames parameter
	// it could be overwritten from ClusterSpec.CMD and cli flags, and it is not required.
	Cmds []string `json:"cmds,omitempty"`

	//LaunchApps This field allows user to specify the app names they want to launch.
	// it could be overwritten from ClusterSpec.APPNames and cli flags.
	LaunchApps []string `json:"launchApps,omitempty"`

	// Configs Additional configurations for the specified app
	//it will override the default launch command and delete command, as well as the corresponding app files.
	Configs []ApplicationConfig `json:"configs,omitempty"`
}

type ApplicationConfig struct {
	// the AppName
	Name string `json:"name,omitempty"`

	// Env is a set of key value pair.
	// it is app level, only this app will be aware of its existence,
	// it is used to render app files, or as an environment variable for app startup and deletion commands
	// it takes precedence over ApplicationSpec.Env.
	Env []string `json:"env,omitempty"`

	//Files indicates that how to modify the specific app files.
	Files []AppFile `json:"files,omitempty"`

	// app Launch customization
	Launch *Launch `json:"launch,omitempty"`

	// app Delete customization
	//Delete *Delete `json:"delete,omitempty"`
}

type Strategy string

const (
	OverWriteStrategy Strategy = "overwrite"
	MergeStrategy     Strategy = "merge"
)

type AppFile struct {
	// Path represents the path to write the Values, required.
	Path string `json:"path,omitempty"`

	//PreProcessor pre mutate the whole Values.
	//PreProcessor string `json:"preProcessor,omitempty"`

	// Enumeration value is "merge", "overwrite", "render". default value is "overwrite".
	// OverWriteStrategy : this will overwrite the FilePath with the Data.
	// MergeStrategy: this will merge the FilePath with the Data, and only yaml files format are supported
	Strategy Strategy `json:"strategy,omitempty"`

	// Data real app launch need.
	// it could be raw content, yaml data, yaml section data, key-value pairs, and so on.
	Data string `json:"data,omitempty"`
}

type Delete struct {
	// raw cmds support
	Cmds []string `json:"cmds,omitempty"`
}

type Launch struct {
	// Cmds raw cmds support, not required, exclusive with app type.
	Cmds []string `json:"cmds,omitempty"`

	// Helm represents the helm app type
	//Helm *Helm `json:"helm,omitempty"`

	// Shell represents the shell app type
	//Shell *Shell `json:"shell,omitempty"`

	// Kube represents the kube app type,
	// The reason why this is an arrays that it can support operations on resources in different namespaces.
	//Kube []Kubectl `json:"kube,omitempty"`
}

type Helm struct {
	// Name will omit the chart values NAME parameter.
	Name string `json:"Name,omitempty"`

	//Chart
	//There are five different ways you can express the chart you want to install:
	//1. By chart reference: helm install mymaria example/mariadb
	//2. By path to a packaged chart: helm install mynginx ./nginx-1.2.3.tgz
	//3. By path to an unpacked chart directory: helm install mynginx ./nginx
	//4. By absolute URL: helm install mynginx https://example.com/charts/nginx-1.2.3.tgz
	//5. By chart reference and repo url: helm install --repo https://example.com/charts/ mynginx nginx
	Chart string `json:"chart,omitempty"`

	//Namespace specifies that where the chart package is installed in
	//it override String to fully override common.names.namespace
	Namespace string

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
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of Application
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Application is the Schema for the application API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
