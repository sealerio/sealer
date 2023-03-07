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

	"golang.org/x/sync/errgroup"
)

func (k *Runtime) reset(mastersToDelete, workersToDelete []net.IP) error {
	if err := k.resetNodes(workersToDelete); err != nil {
		return err
	}

	if err := k.resetMasters(mastersToDelete); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) resetNodes(nodes []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			if err := k.infra.CmdAsync(node, nil, "k0s stop",
				"umount $(df -HT | grep '/var/lib/k0s/kubelet/pods' | awk '{print $7}')",
				"k0s reset",
				"rm -rf /etc/k0s/",
				"rm -rf /usr/bin/k0s",
				"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
				"rm -rf /etc/cni && rm -rf /opt/cni"); err != nil {
				return fmt.Errorf("failed to reset node %s: %v", node, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) resetMasters(nodes []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			if err := k.infra.CmdAsync(node, nil, "k0s stop",
				"k0s reset",
				"rm -rf /etc/k0s/",
				"rm -rf /usr/bin/k0s",
				"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
				"rm -rf /etc/cni && rm -rf /opt/cni",
				"rm -rf .kube/config"); err != nil {
				return fmt.Errorf("failed to reset master %s: %v", node, err)
			}
			return nil
		})
	}
	return eg.Wait()
}
