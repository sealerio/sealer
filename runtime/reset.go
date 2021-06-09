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

package runtime

import (
	"fmt"
	"sync"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

func (d *Default) reset(cluster *v1.Cluster) error {
	err := d.resetNodes(cluster.Spec.Nodes.IPList)
	if err != nil {
		logger.Error("failed to clean nodes %v", err)
	}
	err = d.resetNodes(cluster.Spec.Masters.IPList)
	if err != nil {
		logger.Error("failed to clean masters %v", err)
	}
	err = utils.CleanFiles(common.GetClusterWorkDir(cluster.Name), common.DefaultKubeConfigDir())
	if err != nil {
		// needs continue to clean
		logger.Warn("reset cluster : %v", err)
	}
	return d.RecycleRegistryOnMaster0()
}
func (d *Default) resetNodes(nodes []string) error {
	if len(nodes) == 0 {
		return nil
	}
	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			if err := d.resetNode(node); err != nil {
				logger.Error("delete node %s failed %v", node, err)
			}
		}(node)
	}
	wg.Wait()

	return nil
}
func (d *Default) resetNode(node string) error {
	host := utils.GetHostIP(node)
	if err := d.SSH.CmdAsync(host, fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(d.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, d.APIServer),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(d.Masters[0]))); err != nil {
		return err
	}
	return nil
}
