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

package k0s

import (
	"context"
	"fmt"
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Runtime struct is the runtime interface for k0s
type Runtime struct {
	// cluster is sealer clusterFile
	cluster   *v2.Cluster
	Vlog      int
	RegConfig *registry.Config
}

func (k *Runtime) Init() error {
	return k.init()
}

func (k *Runtime) Upgrade() error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) Reset() error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) JoinMasters(newMastersIPList []net.IP) error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) JoinNodes(newNodesIPList []net.IP) error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) DeleteMasters(mastersIPList []net.IP) error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) DeleteNodes(nodesIPList []net.IP) error {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) GetClusterMetadata() (*runtime.Metadata, error) {
	//TODO implement me
	panic("implement me")
}

func (k *Runtime) UpdateCert(certs []string) error {
	//TODO implement me
	panic("implement me")
}

// NewK0sRuntime arg "clusterConfig" is the k0s config file under etc/${ant_name.yaml}, runtime need read k0s config from it
// Mount image is required before new Runtime.
func NewK0sRuntime(cluster *v2.Cluster) (runtime.Interface, error) {
	return newK0sRuntime(cluster)
}

func newK0sRuntime(cluster *v2.Cluster) (runtime.Interface, error) {
	k := &Runtime{
		cluster: cluster,
	}

	k.RegConfig = registry.GetConfig(k.getImageMountDir(), k.cluster.GetMaster0IP())

	if err := k.checkList(); err != nil {
		return nil, err
	}

	setDebugLevel(k)
	return k, nil
}

func setDebugLevel(k *Runtime) {
	if logrus.GetLevel() == logrus.DebugLevel {
		k.Vlog = 6
	}
}

// checkList do a simple check for cluster spec Hosts which can not be empty.
func (k *Runtime) checkList() error {
	if len(k.cluster.Spec.Hosts) == 0 {
		return fmt.Errorf("hosts spec cannot be empty")
	}
	if k.cluster.GetMaster0IP() == nil {
		return fmt.Errorf("master0 IP cannot be empty")
	}
	return nil
}

// getImageMountDir return a path for mount dir, eg: /var/lib/sealer/data/my-k0s-cluster/mount
func (k *Runtime) getImageMountDir() string {
	return platform.DefaultMountClusterImageDir(k.cluster.Name)
}

// getHostSSHClient return ssh client with destination machine.
func (k *Runtime) getHostSSHClient(hostIP net.IP) (ssh.Interface, error) {
	return ssh.NewStdoutSSHClient(hostIP, k.cluster)
}

// getRootfs return the rootfs dir like: /var/lib/sealer/data/my-k0s-cluster/rootfs
func (k *Runtime) getRootfs() string {
	return common.DefaultTheClusterRootfsDir(k.cluster.Name)
}

// getCertsDir return a Dir value such as: /var/lib/sealer/data/my-k0s-cluster/certs
func (k *Runtime) getCertsDir() string {
	return common.TheDefaultClusterCertDir(k.cluster.Name)
}

// sendFileToHosts sent file to the dst dir on host machine
func (k *Runtime) sendFileToHosts(Hosts []net.IP, src, dst string) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range Hosts {
		node := node
		eg.Go(func() error {
			sshClient, err := k.getHostSSHClient(node)
			if err != nil {
				return fmt.Errorf("failed to send file: %v", err)
			}
			if err := sshClient.Copy(node, src, dst); err != nil {
				return fmt.Errorf("failed to send file: %v", err)
			}
			return err
		})
	}
	return eg.Wait()
}
