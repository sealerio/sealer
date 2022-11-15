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
	"strings"

	"github.com/opencontainers/go-digest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type Layer struct {
	ID    digest.Digest `json:"id,omitempty"` // shaxxx:d6a6c9bfd4ad2901695be1dceca62e1c35a8482982ad6be172fe6958bc4f79d7
	Type  string        `json:"type,omitempty"`
	Value string        `json:"value,omitempty"`
}

// ImageSpec defines the desired state of Image
type ImageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Image. Edit Image_types.go to remove/update
	ID            string      `json:"id,omitempty"`
	Layers        []Layer     `json:"layers,omitempty"`
	SealerVersion string      `json:"sealer_version,omitempty"`
	Platform      Platform    `json:"platform"`
	ImageConfig   ImageConfig `json:"image_config"`
}

// ImageStatus defines the observed state of Image
type ImageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Image is the Schema for the images API
type Image struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	Spec   ImageSpec   `json:"spec,omitempty"  yaml:"spec,omitempty"`
	Status ImageStatus `json:"status,omitempty"  yaml:"status,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ImageList contains a list of Image
type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Image `json:"items,omitempty"`
}

type ImageConfig struct {
	// define this image is application image or normal image.
	ImageType string            `json:"image_type,omitempty"`
	Cmd       ImageCmd          `json:"cmd,omitempty"`
	Args      ImageArg          `json:"args,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

type ImageCmd struct {
	//cmd list of base image
	Parent []string `json:"parent,omitempty"`
	//cmd list of current image
	Current []string `json:"current,omitempty"`
}

type ImageArg struct {
	//arg set of base image
	Parent map[string]string `json:"parent,omitempty"`
	//arg set of current image
	Current map[string]string `json:"current,omitempty"`
}

type Platform struct {
	Architecture string `json:"architecture,omitempty"`
	OS           string `json:"os,omitempty"`
	// Variant is an optional field specifying a variant of the CPU, for
	// example `v7` to specify ARMv7 when architecture is `arm`.
	Variant string `json:"variant,omitempty"`
}

func (p *Platform) ToString() string {
	str := p.OS + "/" + p.Architecture + "/" + p.Variant
	str = strings.TrimSuffix(str, "/")
	str = strings.TrimPrefix(str, "/")
	return str
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
