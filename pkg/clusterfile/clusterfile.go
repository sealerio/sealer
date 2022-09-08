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
	"io/ioutil"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	GetCluster() v2.Cluster
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetKubeadmConfig() *kubeadm_config.KubeadmConfig
}

type ClusterFile struct {
	cluster       v2.Cluster
	configs       []v1.Config
	kubeadmConfig kubeadm_config.KubeadmConfig
	plugins       []v1.Plugin
}

func (c *ClusterFile) GetCluster() v2.Cluster {
	return c.cluster
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
