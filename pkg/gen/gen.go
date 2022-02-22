/*
Copyright © 2022 Alibaba Group Holding Ltd.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package gen

import (
	"fmt"

	v1 "k8s.io/api/core/v1"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/client/k8s"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

const (
	masterLabel = "node-role.kubernetes.io/master"
)

func GenerateClusterfile(name, passwd, image string) error {
	var nodeip, masterip []string
	cluster := &v2.Cluster{}

	cluster.Kind = common.Kind
	cluster.APIVersion = common.APIVersion
	cluster.Name = name
	cluster.Spec.SSH.Passwd = passwd
	cluster.Spec.Image = image

	c, err := k8s.Newk8sClient()
	if err != nil {
		return fmt.Errorf("generate clusterfile failed, %s", err)
	}

	all, err := c.ListNodes()
	if err != nil {
		return fmt.Errorf("generate clusterfile failed, %s", err)
	}
	for _, n := range all.Items {
		for _, v := range n.Status.Addresses {
			if _, ok := n.Labels[masterLabel]; ok {
				if v.Type == v1.NodeInternalIP {
					masterip = append(masterip, v.Address)
				}
			} else if v.Type == v1.NodeInternalIP {
				nodeip = append(nodeip, v.Address)
			}
		}
	}

	masterHosts := v2.Host{
		IPS:   masterip,
		Roles: []string{common.MASTER},
	}

	nodeHosts := v2.Host{
		IPS:   nodeip,
		Roles: []string{common.NODE},
	}

	cluster.Spec.Hosts = append(cluster.Spec.Hosts, masterHosts, nodeHosts)

	fileName := fmt.Sprintf("%s/.sealer/%s/Clusterfile", common.GetHomeDir(), name)
	return utils.MarshalYamlToFile(fileName, cluster)
}
