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

package checker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/utils/mount"

	"github.com/alibaba/sealer/common"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type ApplyChecker struct {
	Desired *v1.Cluster
	Current *v1.Cluster
}

func (a ApplyChecker) Check() error {
	desiredMasters := a.Desired.Spec.Masters
	desiredNodes := a.Desired.Spec.Nodes
	a.Desired.Spec.Masters.IPList = utils.RemoveDuplicate(desiredMasters.IPList)
	a.Desired.Spec.Nodes.IPList = utils.RemoveDuplicate(desiredNodes.IPList)
	//should not be master and node at the same time
	if utils.HasSameHosts(desiredMasters, desiredNodes) {
		return fmt.Errorf("should not be master and node at the same time")
	}
	if a.Current != nil {
		if utils.HasSameHosts(a.Current.Spec.Masters, desiredNodes) ||
			utils.HasSameHosts(a.Current.Spec.Nodes, desiredMasters) {
			return fmt.Errorf("masters nodes do not convert to each other")
		}
		MastersToJoin, MastersToDelete := utils.GetDiffHosts(a.Current.Spec.Masters, a.Desired.Spec.Masters)
		NodesToJoin, NodesToDelete := utils.GetDiffHosts(a.Current.Spec.Nodes, a.Desired.Spec.Nodes)
		//master node should not scale up or down at the same time
		if (len(MastersToDelete) > 0) == (len(NodesToJoin) > 0) ||
			(len(MastersToJoin) > 0) == (len(NodesToDelete) > 0) {
			return fmt.Errorf("should not scale up or down at the same time")
		}
	} else {
		//clean cluster base dir
		baseDir := common.DefaultClusterBaseDir(a.Desired.Name)
		if utils.IsFileExist(baseDir) {
			err := mount.RetryUmountCleanDir(filepath.Join(baseDir, "mount"))
			if err != nil {
				return err
			}
			err = os.RemoveAll(baseDir)
			if err != nil {
				return fmt.Errorf("failed to clean cluster base dir %s: %v, please delete the directory and retry", baseDir, err)
			}
		}
	}
	return nil
}

func NewApplyChecker(current, desired *v1.Cluster) Checker {
	return &ApplyChecker{
		Current: current,
		Desired: desired,
	}
}
