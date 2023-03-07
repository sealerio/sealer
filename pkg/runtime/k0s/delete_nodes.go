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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k *Runtime) deleteNodes(nodesToDelete, remainMasters []net.IP) error {
	var remainMaster0 *net.IP
	if len(remainMasters) > 0 {
		remainMaster0 = &remainMasters[0]
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodesToDelete {
		node := node
		eg.Go(func() error {
			logrus.Infof("Start to delete node %s", node)
			if err := k.deleteNode(node, remainMaster0); err != nil {
				return fmt.Errorf("failed to delete node %s: %v", node, err)
			}
			logrus.Infof("Succeeded in deleting node %s", node)
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) deleteNode(node net.IP, remainMaster0 *net.IP) error {
	/** To delete a node from k0s cluster, following these steps.
	STEP1: drain specified node
	STEP2: stop k0s service
	STEP3: unmount kubelet related pod volume, this would cause k0s reset return error
	STEP4: reset the node with install configuration
	STEP5: remove k0s cluster config generate by k0s under /etc/k0s
	STEP6: remove private registry config in /etc/host
	STEP7: remove bin file such as: kubectl, and remove .kube directory
	STEP8: remove k0s bin file.
	STEP9: delete node though k8s client
	*/
	// remove node, if remainMaster0 is nil, no need delete master from cluster
	if remainMaster0 != nil {
		nodeName, err := k.getNodeName(node)
		if err != nil {
			return err
		}

		if err = k.deleteNodeFromCluster(nodeName); err != nil {
			return err
		}
	}

	remoteCleanCmds := []string{"k0s stop",
		"umount $(df -HT | grep '/var/lib/k0s/kubelet/pods' | awk '{print $7}')",
		"k0s reset",
		"rm -rf /etc/k0s/",
		"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
		"rm -rf /usr/bin/k0s"}
	if err := k.infra.CmdAsync(node, nil, remoteCleanCmds...); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) deleteNodeFromCluster(nodeName string) error {
	client, err := k.GetCurrentRuntimeDriver()
	if err != nil {
		return err
	}
	nodeToDelete := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
	}
	return client.Delete(context.Background(), nodeToDelete)
}
