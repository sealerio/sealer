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

package prune

import (
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

type buildPrune struct {
	pruneRootDir string
}

func NewBuildPrune() Pruner {
	return buildPrune{
		pruneRootDir: common.DefaultTmpDir,
	}
}

func (b buildPrune) Select() ([]string, error) {
	var pruneList []string
	// umount all tmp dir, and delete it
	pruneUnits, err := utils.GetDirNameListInDir(b.pruneRootDir, utils.FilterOptions{
		All:          true,
		WithFullPath: true,
	})
	if err != nil {
		return pruneList, err
	}

	for _, unit := range pruneUnits {
		svc := mount.NewMountServiceByTarget(unit)
		if svc == nil {
			pruneList = append(pruneList, unit)
			continue
		}

		// umount tmp target
		err = svc.TempUMount()
		if err != nil {
			return pruneList, err
		}
		pruneList = append(pruneList, unit)
	}

	return pruneList, nil
}

func (b buildPrune) GetSelectorMessage() string {
	return BuildPruner
}
