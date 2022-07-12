package runtime

import (
	"net"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	// Init exec kubeadm init
	Init(cluster *v2.Cluster) error
	Upgrade() error
	Reset() error
	JoinMasters(newMastersIPList []net.IP) error
	JoinNodes(newNodesIPList []net.IP) error
	DeleteMasters(mastersIPList []net.IP) error
	DeleteNodes(nodesIPList []net.IP) error
	GetClusterMetadata() (*Metadata, error)
	UpdateCert(certs []string) error
}
