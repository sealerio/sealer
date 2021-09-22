// Copyright © 2021 Alibaba Group Holding Ltd.
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

package apply

import (
	"fmt"
	"strconv"

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const MasterRoleLabel = "node-role.kubernetes.io/master"

func (c *DefaultApplier) GetCurrentCluster() (*v1.Cluster, error) {
	return c.getCurrentNodes()
}

func (c *DefaultApplier) getCurrentNodes() (*v1.Cluster, error) {
	nodes, err := c.client.ListNodes()
	if err != nil {
		logger.Warn("%v, will create a new cluster", err)
		return nil, nil
	}

	cluster := &v1.Cluster{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       v1.ClusterSpec{},
		Status:     v1.ClusterStatus{},
	}

	for _, node := range nodes.Items {
		addr := getNodeAddress(&node)
		if addr == "" {
			continue
		}
		if _, ok := node.Labels[MasterRoleLabel]; ok {
			cluster.Spec.Masters.IPList = append(cluster.Spec.Masters.IPList, addr)
			continue
		}
		cluster.Spec.Nodes.IPList = append(cluster.Spec.Nodes.IPList, addr)
	}
	cluster.Spec.Masters.Count = strconv.Itoa(len(cluster.Spec.Masters.IPList))
	cluster.Spec.Nodes.Count = strconv.Itoa(len(cluster.Spec.Nodes.IPList))

	return cluster, nil
}

func getNodeAddress(node *corev1.Node) string {
	if len(node.Status.Addresses) < 1 {
		return ""
	}
	return node.Status.Addresses[0].Address
}

func (c *DefaultApplier) DeleteNodes(nodeIPs []string) error {
	logger.Info("delete nodes %s", nodeIPs)
	nodes, err := c.client.ListNodes()
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		addr := getNodeAddress(&node)
		if addr == "" || utils.NotIn(addr, nodeIPs) {
			continue
		}
		if err := c.client.DeleteNode(node.Name); err != nil {
			return fmt.Errorf("failed to delete node %v", err)
		}
	}
	return nil
}
