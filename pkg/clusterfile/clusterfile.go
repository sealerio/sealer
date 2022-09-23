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
	"path/filepath"

	"os"

	"github.com/sealerio/sealer/common"
	utilsos "github.com/sealerio/sealer/utils/os"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadmconfig"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	GetCluster() v2.Cluster
	SetCluster(v2.Cluster)
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetKubeadmConfig() *kubeadmconfig.KubeadmConfig
	SaveAll() error
}

type ClusterFile struct {
	cluster       *v2.Cluster
	configs       []v1.Config
	kubeadmConfig kubeadmconfig.KubeadmConfig
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

func (c *ClusterFile) GetKubeadmConfig() *kubeadmconfig.KubeadmConfig {
	return &c.kubeadmConfig
}

func (c *ClusterFile) SaveAll() error {
	var (
		clusterfileBytes [][]byte
		config           []byte
		plugin           []byte
	)
	fileName := common.GetDefaultClusterfile()
	err := os.MkdirAll(filepath.Dir(fileName), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to mkdir %s: %v", fileName, err)
	}

	cluster, err := yaml.Marshal(c.cluster)
	if err != nil {
		return err
	}
	clusterfileBytes = append(clusterfileBytes, cluster)

	if len(c.configs) != 0 {
		for _, cg := range c.configs {
			config, err = yaml.Marshal(cg)
			if err != nil {
				return err
			}
			clusterfileBytes = append(clusterfileBytes, config)
		}
	}

	if len(c.plugins) != 0 {
		for _, p := range c.plugins {
			plugin, err = yaml.Marshal(p)
			if err != nil {
				return err
			}
			clusterfileBytes = append(clusterfileBytes, plugin)
		}
	}

	if len(c.kubeadmConfig.InitConfiguration.TypeMeta.Kind) != 0 {
		initConfiguration, err := yaml.Marshal(c.kubeadmConfig.InitConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, initConfiguration)
	}

	if c.kubeadmConfig.JoinConfiguration != nil {
		joinConfiguration, err := yaml.Marshal(c.kubeadmConfig.JoinConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, joinConfiguration)
	}

	if c.kubeadmConfig.ClusterConfiguration != nil {
		clusterConfiguration, err := yaml.Marshal(c.kubeadmConfig.ClusterConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, clusterConfiguration)
	}

	if c.kubeadmConfig.KubeletConfiguration != nil {
		kubeletConfiguration, err := yaml.Marshal(c.kubeadmConfig.KubeletConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, kubeletConfiguration)
	}

	if c.kubeadmConfig.KubeProxyConfiguration != nil {
		kubeProxyConfiguration, err := yaml.Marshal(c.kubeadmConfig.KubeProxyConfiguration)
		if err != nil {
			return err
		}
		clusterfileBytes = append(clusterfileBytes, kubeProxyConfiguration)
	}
	//todo cluster ssh info need to be encrypted

	return utilsos.NewCommonWriter(fileName).WriteFile(bytes.Join(clusterfileBytes, []byte("---\n")))
}

func NewClusterFile(b []byte) (Interface, error) {
	clusterFile := new(ClusterFile)
	if err := decodeClusterFile(bytes.NewReader(b), clusterFile); err != nil {
		return nil, fmt.Errorf("failed to load clusterfile: %v", err)
	}

	return clusterFile, nil
}
