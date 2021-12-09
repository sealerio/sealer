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

package applydriver

import (
	"fmt"

	"github.com/alibaba/sealer/client/k8s"
	"github.com/alibaba/sealer/common"
	v2 "github.com/alibaba/sealer/types/api/v2"
	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

const MasterRoleLabel = "node-role.kubernetes.io/master"

func GetCurrentCluster(client *k8s.Client) (*v2.Cluster, error) {
	if client == nil {
		return nil, nil
	}
	nodes, err := client.ListNodes()
	if err != nil {
		return nil, err
	}

	cluster := &v2.Cluster{}
	masterIPList := []string{}
	nodeIPList := []string{}

	for _, node := range nodes.Items {
		addr := getNodeAddress(&node)
		if addr == "" {
			continue
		}
		if _, ok := node.Labels[MasterRoleLabel]; ok {
			masterIPList = append(masterIPList, addr)
			continue
		}
		nodeIPList = append(nodeIPList, addr)
	}
	cluster.Spec.Hosts = []v2.Host{{IPS: masterIPList, Roles: []string{common.MASTER}}, {IPS: nodeIPList, Roles: []string{common.NODE}}}

	return cluster, nil
}

func DeleteNodes(client *k8s.Client, nodeIPs []string) error {
	logger.Info("delete nodes %s", nodeIPs)
	nodes, err := client.ListNodes()
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		addr := getNodeAddress(&node)
		if addr == "" || utils.NotIn(addr, nodeIPs) {
			continue
		}
		if err := client.DeleteNode(node.Name); err != nil {
			return fmt.Errorf("failed to delete node %v", err)
		}
	}
	return nil
}

func getNodeAddress(node *corev1.Node) string {
	if len(node.Status.Addresses) < 1 {
		return ""
	}
	return node.Status.Addresses[0].Address
}
