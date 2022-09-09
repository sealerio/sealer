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
	"errors"
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
	"sync"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utils_os "github.com/sealerio/sealer/utils/os"
)

var ErrTypeNotFound = errors.New("no corresponding type structure was found")

type ClusterFile struct {
	path       string
	Cluster    v2.Cluster
	Configs    []v1.Config
	KubeConfig *kubeadm_config.KubeadmConfig
	Plugins    []v1.Plugin
}

var (
	clusterFile = &ClusterFile{}
	once        sync.Once
)

type Interface interface {
	PreProcessor
	GetCluster() *v2.Cluster
	SetCluster(*v2.Cluster)
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetKubeadmConfig() *kubeadm_config.KubeadmConfig
	SaveAll() error
}

func (c *ClusterFile) GetCluster() *v2.Cluster {
	return &c.Cluster
}

func (c *ClusterFile) SetCluster(cluster *v2.Cluster) {
	c.Cluster = *cluster
}

func (c *ClusterFile) GetConfigs() []v1.Config {
	return c.Configs
}

func (c *ClusterFile) GetPlugins() []v1.Plugin {
	return c.Plugins
}

func (c *ClusterFile) GetKubeadmConfig() *kubeadm_config.KubeadmConfig {
	return c.KubeConfig
}

func (c *ClusterFile) SaveAll() error {
	var configs [][]byte

	cluster, err := yaml.Marshal(c.Cluster)
	if err != nil {
		return err
	}

	config, err := yaml.Marshal(c.Configs)
	if err != nil {
		return err
	}

	plugin, err := yaml.Marshal(c.Plugins)
	if err != nil {
		return err
	}

	kubeconfig, err := yaml.Marshal(c.KubeConfig)
	if err != nil {
		return err
	}

	configs = append(configs, cluster, config, plugin, kubeconfig)

	path := common.GetClusterWorkClusterfile()

	if err := utils_os.NewCommonWriter(path).WriteFile(bytes.Join(configs, []byte("---\n"))); err != nil {
		return err
	}

	return nil
}

func NewClusterFile(path string) (i Interface, err error) {
	if !filepath.IsAbs(path) {
		pa, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(pa, path)
	}

	if path == "" {
		return clusterFile, nil
	}
	once.Do(func() {
		clusterFile.path = path
		err = clusterFile.Process()
	})
	return clusterFile, err
}
