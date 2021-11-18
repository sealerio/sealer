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
	Upgrade(cluster *v2.Cluster) error
	Reset(cluster *v2.Cluster) error
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
	*Metadata
	*v2.Cluster
	*KubeadmConfig
	*RuntimeConfig
}

func (k *KubeadmRuntime) Init(cluster *v2.Cluster) error {
	return k.init(cluster)
}

func (k *KubeadmRuntime) Upgrade(cluster *v2.Cluster) error {
	panic("implement me")
}

func (k *KubeadmRuntime) Reset(cluster *v2.Cluster) error {
	panic("implement me")
}

func (k *KubeadmRuntime) JoinMasters(newMastersIPList []string) error {
	panic("implement me")
}

func (k *KubeadmRuntime) JoinNodes(newNodesIPList []string) error {
	panic("implement me")
}

func (k *KubeadmRuntime) DeleteMasters(mastersIPList []string) error {
	panic("implement me")
}

func (k *KubeadmRuntime) DeleteNodes(nodesIPList []string) error {
	panic("implement me")
}

// clusterfile is the Clusterfile path/name, runtime need read kubeadm config from it
func NewDefaultRuntime(cluster *v2.Cluster, clusterfile string) (Interface, error) {
	return newKubeadmRuntime(cluster, clusterfile)
}
