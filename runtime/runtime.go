package runtime

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	// exec kubeadm init
	LoadMetadata()
	Init(cluster *v1.Cluster) error
	Hook(cluster *v1.Cluster) error
	Upgrade(cluster *v1.Cluster) error
	Reset(cluster *v1.Cluster) error
	CNI(cluster *v1.Cluster) error
	JoinMasters(newMastersIPList []string) error
	JoinNodes(newNodesIPList []string) error
	DeleteMasters(mastersIPList []string) error
	DeleteNodes(nodesIPList []string) error
	CopyFilesOnRoot(cluster *v1.Cluster) error
	CleanFilesOnRoot(cluster *v1.Cluster) error
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

	// net config
	Interface  string
	Network    string
	CIDR       string
	IPIP       bool
	MTU        string
	WithoutCNI bool
}

func NewDefaultRuntime(cluster *v1.Cluster) Interface {
	d := &Default{}
	err := d.initRunner(cluster)
	if err != nil {
		return nil
	}
	return d
}

func (d *Default) LoadMetadata() {
	metadataPath := filepath.Join("/tmp", d.ClusterName, common.DefaultMetadataName)
	var metadataFile []byte
	var err error
	metadata := &Metadata{}
	if utils.IsFileExist(metadataPath) {
		metadataFile, err = ioutil.ReadFile(metadataPath)
		if err != nil {
			logger.Warn("read metadata is error: %v", err)
		}
		err = json.Unmarshal(metadataFile, metadata)
		if err != nil {
			logger.Warn("load metadata failed, skip")
			return
		}
		logger.Debug("metadata version %s", metadata.Version)
	}
	d.Metadata = metadata
}
func (d *Default) Reset(cluster *v1.Cluster) error {
	return d.reset(cluster)
}

func (d *Default) CNI(cluster *v1.Cluster) error {
	return d.cni(cluster)
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
