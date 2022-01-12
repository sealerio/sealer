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
	"os"
	"sync"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"

	v2 "github.com/alibaba/sealer/types/api/v2"
)

type Interface interface {
	// Init exec kubeadm init
	Init(cluster *v2.Cluster) error
	Upgrade() error
	Reset() error
	JoinMasters(newMastersIPList []string) error
	JoinNodes(newNodesIPList []string) error
	DeleteMasters(mastersIPList []string) error
	DeleteNodes(nodesIPList []string) error
	GetClusterMetadata() (*Metadata, error)
}

type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	//KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion"`
}

type KubeadmRuntime struct {
	*sync.Mutex
	*v2.Cluster
	*KubeadmConfig
	*Config
}

var ForceDelete bool

func (k *KubeadmRuntime) Init(cluster *v2.Cluster) error {
	return k.init(cluster)
}

func (k *KubeadmRuntime) Upgrade() error {
	return k.upgrade()
}

func (k *KubeadmRuntime) Reset() error {
	logger.Info("Start to delete cluster: master %s, node %s", k.Cluster.GetMasterIPList(), k.Cluster.GetNodeIPList())
	if err := k.confirmDeleteNodes(); err != nil {
		return err
	}
	return k.reset()
}

func (k *KubeadmRuntime) JoinMasters(newMastersIPList []string) error {
	if len(newMastersIPList) != 0 {
		logger.Info("%s will be added as master", newMastersIPList)
	}
	return k.joinMasters(newMastersIPList)
}

func (k *KubeadmRuntime) JoinNodes(newNodesIPList []string) error {
	if len(newNodesIPList) != 0 {
		logger.Info("%s will be added as worker", newNodesIPList)
	}
	return k.joinNodes(newNodesIPList)
}

func (k *KubeadmRuntime) DeleteMasters(mastersIPList []string) error {
	if len(mastersIPList) != 0 {
		logger.Info("master %s will be deleted", mastersIPList)
		if err := k.confirmDeleteNodes(); err != nil {
			return err
		}
	}
	return k.deleteMasters(mastersIPList)
}

func (k *KubeadmRuntime) DeleteNodes(nodesIPList []string) error {
	if len(nodesIPList) != 0 {
		logger.Info("worker %s will be deleted", nodesIPList)
		if err := k.confirmDeleteNodes(); err != nil {
			return err
		}
	}
	return k.deleteNodes(nodesIPList)
}

func (k *KubeadmRuntime) confirmDeleteNodes() error {
	if !ForceDelete {
		if pass, err := utils.ConfirmOperation("Are you sure to delete these nodes? "); err != nil {
			return err
		} else if !pass {
			os.Exit(0)
		}
	}
	return nil
}

func (k *KubeadmRuntime) GetClusterMetadata() (*Metadata, error) {
	return k.getClusterMetadata()
}

// NewDefaultRuntime arg "clusterfile" is the Clusterfile path/name, runtime need read kubeadm config from it
func NewDefaultRuntime(cluster *v2.Cluster, clusterfile string) (Interface, error) {
	return newKubeadmRuntime(cluster, clusterfile)
}
