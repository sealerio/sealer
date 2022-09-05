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

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) reset() error {
	k.resetNodes(k.cluster.GetNodeIPList())
	k.resetMasters(k.cluster.GetMasterIPList())

	return k.DeleteRegistry()
}

func (k *Runtime) resetNodes(nodes []net.IP) {
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			if err := k.resetNode(node); err != nil {
				logrus.Errorf("failed to delete node %s: %v", node, err)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return
	}
}

func (k *Runtime) resetMasters(nodes []net.IP) {
	for _, node := range nodes {
		if err := k.resetNode(node); err != nil {
			logrus.Errorf("failed to delete master(%s): %v", node, err)
		}
	}
}

func (k *Runtime) resetNode(node net.IP) error {
	ssh, err := k.getHostSSHClient(node)
	if err != nil {
		return fmt.Errorf("failed to reset node: %v", err)
	}
	if err := ssh.CmdAsync(node, fmt.Sprintf(RemoteCleanMasterOrNode, DefaultK0sConfigPath, ExternalCRI),
		RemoveKubeConfig,
		fmt.Sprintf(RemoteRemoveEtcHost, SeaHub),
		fmt.Sprintf(RemoteRemoveEtcHost, k.RegConfig.Domain),
		fmt.Sprintf(RemoteRemoveRegistryCerts, k.RegConfig.Domain),
		fmt.Sprintf(RemoteRemoveRegistryCerts, SeaHub)); err != nil {
		return err
	}
	return nil
}
