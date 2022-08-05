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
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/k0sproject/k0sctl/pkg/apis/k0sctl.k0sproject.io/v1beta1/cluster"
)

// APIVersion is the current api version
const APIVersion = "k0sctl.k0sproject.io/v1beta1"

// ClusterMetadata defines cluster metadata
type ClusterMetadata struct {
	Name       string `yaml:"name" validate:"required" default:"k0s-cluster"`
	Kubeconfig string `yaml:"-"`
}

// Cluster describes launchpad.yaml configuration
type Cluster struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Metadata   *ClusterMetadata `yaml:"metadata"`
	Spec       *cluster.Spec    `yaml:"spec"`
}

// UnmarshalYAML sets in some sane defaults when unmarshaling the data from yaml
func (c *Cluster) UnmarshalYAML(unmarshal func(interface{}) error) error {
	c.Metadata = &ClusterMetadata{
		Name: "k0s-cluster",
	}
	c.Spec = &cluster.Spec{}

	type clusterConfig Cluster
	yc := (*clusterConfig)(c)

	if err := unmarshal(yc); err != nil {
		return err
	}

	return nil
}

// Validate performs a configuration sanity check
func (c *Cluster) Validate() error {
	validation.ErrorTag = "yaml"
	return validation.ValidateStruct(c,
		validation.Field(&c.APIVersion, validation.Required, validation.In(APIVersion).Error("must equal "+APIVersion)),
		validation.Field(&c.Kind, validation.Required, validation.In("cluster", "Cluster").Error("must equal Cluster")),
		validation.Field(&c.Spec),
	)
}
