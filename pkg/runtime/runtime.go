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
	"fmt"
	"sync"

	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sirupsen/logrus"
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
	UpdateCert(certs []string) error
}

type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
	Variant string `json:"variant"`
	//ClusterRuntime is a Flag to distinguish a cluster install tool.
	ClusterRuntime string `json:"cluster_runtime"`
	//KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion"`
	NydusFlag   bool   `json:"NydusFlag"`
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
	logrus.Infof("Start to delete cluster: master %s, node %s", k.Cluster.GetMasterIPList(), k.Cluster.GetNodeIPList())
	if err := k.confirmDeleteNodes(); err != nil {
		return err
	}
	return k.reset()
}

func (k *KubeadmRuntime) JoinMasters(newMastersIPList []string) error {
	if len(newMastersIPList) != 0 {
		logrus.Infof("%s will be added as master", newMastersIPList)
	}
	return k.joinMasters(newMastersIPList)
}

func (k *KubeadmRuntime) JoinNodes(newNodesIPList []string) error {
	if len(newNodesIPList) != 0 {
		logrus.Infof("%s will be added as worker", newNodesIPList)
	}
	return k.joinNodes(newNodesIPList)
}

func (k *KubeadmRuntime) DeleteMasters(mastersIPList []string) error {
	if len(mastersIPList) != 0 {
		logrus.Infof("master %s will be deleted", mastersIPList)
		if err := k.confirmDeleteNodes(); err != nil {
			return err
		}
	}
	return k.deleteMasters(mastersIPList)
}

func (k *KubeadmRuntime) DeleteNodes(nodesIPList []string) error {
	if len(nodesIPList) != 0 {
		logrus.Infof("worker %s will be deleted", nodesIPList)
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
			return fmt.Errorf("exit the operation of delete these nodes")
		}
	}
	return nil
}

func (k *KubeadmRuntime) GetClusterMetadata() (*Metadata, error) {
	return k.getClusterMetadata()
}

func (k *KubeadmRuntime) UpdateCert(certs []string) error {
	return k.updateCert(certs)
}

// NewDefaultRuntime arg "clusterfileKubeConfig" is the Clusterfile path/name, runtime need read kubeadm config from it
// Mount image is required before new Runtime.
func NewDefaultRuntime(cluster *v2.Cluster, clusterfileKubeConfig *KubeadmConfig) (Interface, error) {
	return newKubeadmRuntime(cluster, clusterfileKubeConfig)
}
