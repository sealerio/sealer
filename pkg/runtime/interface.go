// Copyright © 2022 Alibaba Group Holding Ltd.
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
	"net"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

type Interface interface {
	// Init exec init phase for cluster, v2.Cluster deliver cluster's information.
	Init(cluster *v2.Cluster) error
	// Upgrade exec upgrading phase for cluster.
	Upgrade() error
	// Reset exec reset phase for cluster.
	Reset() error
	// JoinMasters exec joining phase for cluster, add master role for these nodes. net.IP is the master node IP array.
	JoinMasters(newMastersIPList []net.IP) error
	// JoinNodes exec joining phase for cluster, add worker/<none> role for these nodes. net.IP is the worker/<none> node IP array.
	JoinNodes(newNodesIPList []net.IP) error
	// DeleteMasters exec deleting phase for deleting cluster master role nodes. net.IP is the master node IP array.
	DeleteMasters(mastersIPList []net.IP) error
	// DeleteNodes exec deleting phase for deleting worker/<none> master role nodes. net.IP is the worker/<none> node IP array.
	DeleteNodes(nodesIPList []net.IP) error
	// GetClusterMetadata read the rootfs/Metadata file to get some install info for cluster.
	GetClusterMetadata() (*Metadata, error)
	// UpdateCert exec Update certs phase for renew k8s cluster's certs such as: etcd/apiServer, It seems unnecessary for k0s、k3s.
	UpdateCert(certs []string) error
}
