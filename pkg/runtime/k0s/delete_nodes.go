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
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) deleteNodes(nodes []net.IP) error {
	if len(nodes) == 0 {
		return nil
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			logrus.Infof("Start to delete worker %s", node)
			if err := k.deleteNode(node); err != nil {
				return fmt.Errorf("failed to delete node %s: %v", node, err)
			}
			logrus.Infof("Succeeded in deleting worker %s", node)
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) deleteNode(node net.IP) error {
	ssh, err := k.getHostSSHClient(node)
	if err != nil {
		return fmt.Errorf("failed to delete node: %v", err)
	}
	remoteCleanCmds := []string{fmt.Sprintf(RemoteCleanMasterOrNode, DefaultK0sConfigPath, ExternalCRI),
		fmt.Sprintf(RemoteRemoveEtcHost, k.RegConfig.Domain),
		fmt.Sprintf(RemoteRemoveEtcHost, SeaHub),
		fmt.Sprintf(RemoteRemoveRegistryCerts, k.RegConfig.Domain),
		fmt.Sprintf(RemoteRemoveRegistryCerts, SeaHub),
		RemoveKubeConfig,
		RemoveK0sBin}
	if err := ssh.CmdAsync(node, remoteCleanCmds...); err != nil {
		return err
	}

	//remove node
	if len(k.cluster.GetMasterIPList()) > 0 {
		hostname, err := k.isHostName(k.cluster.GetMaster0IP(), node)
		if err != nil {
			return err
		}
		ssh, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
		if err != nil {
			return fmt.Errorf("failed to get master0 ssh client(%s): %v", k.cluster.GetMaster0IP(), err)
		}
		if err := ssh.CmdAsync(k.cluster.GetMaster0IP(), fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("failed to delete node %s: %v", hostname, err)
		}
	}

	return nil
}
