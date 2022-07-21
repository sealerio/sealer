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
	"errors"
	"sync"

	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

var ErrTypeNotFound = errors.New("no corresponding type structure was found")

type ClusterFile struct {
	path       string
	Cluster    v2.Cluster
	Configs    []v1.Config
	KubeConfig *kubeadm.KubeadmConfig
	Plugins    []v1.Plugin
}

var (
	clusterFile = &ClusterFile{}
	once        sync.Once
)

type Interface interface {
	PreProcessor
	GetCluster() v2.Cluster
	GetConfigs() []v1.Config
	GetPlugins() []v1.Plugin
	GetKubeadmConfig() *kubeadm.KubeadmConfig
}

func (c *ClusterFile) GetCluster() v2.Cluster {
	return c.Cluster
}

func (c *ClusterFile) GetConfigs() []v1.Config {
	return c.Configs
}

func (c *ClusterFile) GetPlugins() []v1.Plugin {
	return c.Plugins
}

func (c *ClusterFile) GetKubeadmConfig() *kubeadm.KubeadmConfig {
	return c.KubeConfig
}

func NewClusterFile(path string) (i Interface, err error) {
	if path == "" {
		return clusterFile, nil
	}
	once.Do(func() {
		clusterFile.path = path
		err = clusterFile.Process()
	})
	return clusterFile, err
}
