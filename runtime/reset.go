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

	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func (d *Default) reset(cluster *v1.Cluster) error {
	d.resetNodes(cluster.Spec.Nodes.IPList)
	d.resetMasters(cluster.Spec.Masters.IPList)
	return d.RecycleRegistry()
}

func (d *Default) resetNodes(nodes []string) {
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
}

func (d *Default) resetMasters(nodes []string) {
	for _, node := range nodes {
		if err := d.resetNode(node); err != nil {
			logger.Error("delete master %s failed %v", node, err)
		}
	}
}

func (d *Default) resetNode(node string) error {
	if err := d.SSH.CmdAsync(node, fmt.Sprintf(RemoteCleanMasterOrNode, vlogToStr(d.Vlog)),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, d.APIServer),
		fmt.Sprintf(RemoteRemoveAPIServerEtcHost, getRegistryHost(d.Rootfs, d.Masters[0]))); err != nil {
		return err
	}
	return nil
}
