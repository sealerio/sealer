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

func (k *Runtime) reset() error {
	if err := k.resetNodes(k.cluster.GetNodeIPList()); err != nil {
		return err
	}
	if err := k.resetMasters(k.cluster.GetMasterIPList()); err != nil {
		return err
	}

	return k.DeleteRegistry()
}

func (k *Runtime) resetNodes(nodes []net.IP) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			if err := k.resetNode(node); err != nil {
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
			if err := k.resetNode(node); err != nil {
				return fmt.Errorf("failed to reset master %s: %v", node, err)
			}
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) resetNode(node net.IP) error {
	ssh, err := k.getHostSSHClient(node)
	if err != nil {
		return err
	}

	/** To reset a node, do following commands one by one:
	STEP1: stop k0s service
	STEP2: reset the node with install configuration
	STEP3: remove k0s cluster config generate by k0s under /etc/k0s
	STEP4: remove private registry config in /etc/host
	*/
	if err := ssh.CmdAsync(node, "k0s stop",
		fmt.Sprintf("k0s reset --cri-socket %s", ExternalCRI),
		"rm -rf /etc/k0s/",
		"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", SeaHub),
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, SeaHub)); err != nil {
		return err
	}
	return nil
}
