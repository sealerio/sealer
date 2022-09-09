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
	"io/ioutil"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"sync"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utils_os "github.com/sealerio/sealer/utils/os"
)

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

func NewClusterFile(path string) (Interface, error) {
	clusterFileData, err := ioutil.ReadFile(filepath.Clean(path))

	if err != nil {
		return nil, err
	}

	clusterFile := new(ClusterFile)
	err = decodeClusterFile(bytes.NewReader(clusterFileData), clusterFile)

	if err != nil {
		return nil, fmt.Errorf("failed to load clusterfile: %v", err)
	}

	return clusterFile, nil
}
