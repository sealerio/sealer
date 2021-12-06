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

package runtime

import (
	v2 "github.com/alibaba/sealer/types/api/v2"
)

type Interface interface {
	// exec kubeadm init
	Init(cluster *v2.Cluster) error
	Upgrade() error
	Reset() error
	JoinMasters(newMastersIPList []string) error
	JoinNodes(newNodesIPList []string) error
	DeleteMasters(mastersIPList []string) error
	DeleteNodes(nodesIPList []string) error
}

type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

type KubeadmRuntime struct {
	*v2.Cluster
	*KubeadmConfig
	*Config
}

func (k *KubeadmRuntime) Init(cluster *v2.Cluster) error {
	return k.init(cluster)
}

func (k *KubeadmRuntime) Upgrade() error {
	return k.upgrade()
}

func (k *KubeadmRuntime) Reset() error {
	return k.reset()
}

func (k *KubeadmRuntime) JoinMasters(newMastersIPList []string) error {
	return k.joinMasters(newMastersIPList)
}

func (k *KubeadmRuntime) JoinNodes(newNodesIPList []string) error {
	return k.joinNodes(newNodesIPList)
}

func (k *KubeadmRuntime) DeleteMasters(mastersIPList []string) error {
	return k.deleteMasters(mastersIPList)
}

func (k *KubeadmRuntime) DeleteNodes(nodesIPList []string) error {
	return k.deleteNodes(nodesIPList)
}

// clusterfile is the Clusterfile path/name, runtime need read kubeadm config from it
func NewDefaultRuntime(cluster *v2.Cluster, clusterfile string) (Interface, error) {
	return newKubeadmRuntime(cluster, clusterfile)
}
