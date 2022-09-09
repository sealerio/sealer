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

package clusterfile

import (
	"bytes"
	"fmt"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/utils/os"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	GetCluster() v2.Cluster
	SetCluster(v2.Cluster)
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetKubeadmConfig() *kubeadm_config.KubeadmConfig
	SaveAll() error
}

type ClusterFile struct {
	cluster       *v2.Cluster
	configs       []v1.Config
	kubeadmConfig kubeadm_config.KubeadmConfig
	plugins       []v1.Plugin
}

func (c *ClusterFile) GetCluster() v2.Cluster {
	return *c.cluster
}

func (c *ClusterFile) SetCluster(cluster v2.Cluster) {
	c.cluster = &cluster
}

func (c *ClusterFile) GetConfigs() []v1.Config {
	return c.configs
}

func (c *ClusterFile) GetPlugins() []v1.Plugin {
	return c.plugins
}

func (c *ClusterFile) GetKubeadmConfig() *kubeadm_config.KubeadmConfig {
	return &c.kubeadmConfig
}

func (c *ClusterFile) SaveAll() error {
	var configs [][]byte

	cluster, err := yaml.Marshal(c.cluster)
	if err != nil {
		return err
	}

	config, err := yaml.Marshal(c.configs)
	if err != nil {
		return err
	}

	plugin, err := yaml.Marshal(c.plugins)
	if err != nil {
		return err
	}

	kubeconfig, err := yaml.Marshal(c.kubeadmConfig)
	if err != nil {
		return err
	}

	configs = append(configs, cluster, config, plugin, kubeconfig)

	path := common.GetClusterWorkClusterfile()

	if err := os.NewCommonWriter(path).WriteFile(bytes.Join(configs, []byte("---\n"))); err != nil {
		return err
	}

	return nil
}

func NewClusterFile(b []byte) (Interface, error) {
	clusterFile := new(ClusterFile)
	if err := decodeClusterFile(bytes.NewReader(b), clusterFile); err != nil {
		return nil, fmt.Errorf("failed to load clusterfile: %v", err)
	}

	return clusterFile, nil
}
