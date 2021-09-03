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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/image/store"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	// exec kubeadm init
	Init(cluster *v1.Cluster) error
	Hook(cluster *v1.Cluster) error
	Upgrade(cluster *v1.Cluster) error
	Reset(cluster *v1.Cluster) error
	JoinMasters(newMastersIPList []string) error
	JoinNodes(newNodesIPList []string) error
	DeleteMasters(mastersIPList []string) error
	DeleteNodes(nodesIPList []string) error
}

type Metadata struct {
	Version string `json:"version"`
	Arch    string `json:"arch"`
}

type Default struct {
	Metadata          *Metadata
	ClusterName       string
	Token             string
	APIServerCertSANs []string
	SvcCIDR           string
	PodCIDR           string
	ControlPlaneRepo  string
	RegistryPort      int
	DNSDomain         string
	Masters           []string
	APIServer         string
	CertPath          string
	StaticFileDir     string
	CertEtcdPath      string
	JoinToken         string
	VIP               string
	EtcdDevice        string
	KubeadmFilePath   string
	TokenCaCertHash   string
	CertificateKey    string
	Vlog              int
	Nodes             []string
	LvscareImage      string
	SSH               ssh.Interface
	Rootfs            string
	BasePath          string
	imageStore        store.ImageStore
	CriSocket         string
	CriCGroupDriver   string
	KubeadmApi        string
}

func NewDefaultRuntime(cluster *v1.Cluster) Interface {
	c, err := store.NewDefaultImageStore()
	if err != nil {
		return nil
	}
	d := &Default{imageStore: c}
	err = d.initRunner(cluster)
	if err != nil {
		logger.Error("get runtime failed %v", err)
		return nil
	}
	return d
}

func (d *Default) LoadMetadata() error {
	metadataPath := filepath.Join(common.DefaultMountCloudImageDir(d.ClusterName), common.DefaultMetadataName)
	var metadataFile []byte
	var err error
	metadata := &Metadata{}
	metadataFile, err = ioutil.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read CloudImage metadata %v", err)
	}
	err = json.Unmarshal(metadataFile, metadata)
	if err != nil {
		return fmt.Errorf("failed to load CloudImage metadata %v", err)
	}
	logger.Info("metadata version %s", metadata.Version)
	d.Metadata = metadata
	return nil
}
func (d *Default) Reset(cluster *v1.Cluster) error {
	return d.reset(cluster)
}

func (d *Default) Upgrade(cluster *v1.Cluster) error {
	panic("implement upgrade !!")
}

func (d *Default) JoinMasters(newMastersIPList []string) error {
	logger.Debug("join masters: %v", newMastersIPList)
	return d.joinMasters(newMastersIPList)
}

func (d *Default) JoinNodes(newNodesIPList []string) error {
	logger.Debug("join nodes: %v", newNodesIPList)
	return d.joinNodes(newNodesIPList)
}

func (d *Default) DeleteMasters(mastersIPList []string) error {
	logger.Debug("delete masters: %v", mastersIPList)
	return d.deleteMasters(mastersIPList)
}

func (d *Default) DeleteNodes(nodesIPList []string) error {
	logger.Debug("delete nodes: %v", nodesIPList)
	return d.deleteNodes(nodesIPList)
}

func (d *Default) Init(cluster *v1.Cluster) error {
	return d.init(cluster)
}

func (d *Default) Hook(cluster *v1.Cluster) error {
	panic("implement me")
}
