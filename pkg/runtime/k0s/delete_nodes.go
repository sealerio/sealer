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

	"github.com/sealerio/sealer/pkg/client/k8s"

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
			logrus.Infof("Start to delete node %s", node)
			if err := k.deleteNode(node); err != nil {
				return fmt.Errorf("failed to delete node %s: %v", node, err)
			}
			logrus.Infof("Succeeded in deleting node %s", node)
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
	/** To delete a node from k0s cluster, following these steps.
	STEP1: stop k0s service
	STEP2: reset the node with install configuration
	STEP3: remove k0s cluster config generate by k0s under /etc/k0s
	STEP4: remove private registry config in /etc/host
	STEP5: remove bin file such as: kubectl, and remove .kube directory
	STEP6: remove k0s bin file.
	STEP7: delete node though k8s client
	*/
	remoteCleanCmds := []string{"k0s stop",
		fmt.Sprintf("k0s reset --config %s --cri-socket %s", DefaultK0sConfigPath, ExternalCRI),
		"rm -rf /etc/k0s/",
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", SeaHub),
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, SeaHub),
		"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
		"rm -rf /usr/bin/k0s"}
	if err := ssh.CmdAsync(node, remoteCleanCmds...); err != nil {
		return err
	}

	//remove node
	if len(k.cluster.GetMasterIPList()) > 0 {
		hostname, err := k.isHostName(node)
		if err != nil {
			return err
		}
		client, err := k8s.Newk8sClient()
		if err != nil {
			return err
		}
		if err := client.DeleteNode(strings.TrimSpace(hostname)); err != nil {
			return err
		}
	}

	return nil
}
