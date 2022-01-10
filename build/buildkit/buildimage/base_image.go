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

package buildimage

import (
	"fmt"
	"strings"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/mount"
)

// GetLayerMountInfo to get rootfs mount info.
//1, already mount: runtime docker registry mount info,just get related mount info.
//2, already mount: if exec build cmd failed and return ,need to collect related old mount info
//3, new mount: just mount and return related info.
func GetLayerMountInfo(baseLayers []v1.Layer, buildType string) (*buildinstruction.MountTarget, error) {
	filter := map[string]string{
		common.LocalBuild: "rootfs",
		common.LiteBuild:  "tmp",
	}
	mountInfos := mount.GetBuildMountInfo(filter[buildType])

	if buildType == common.LocalBuild {
		if len(mountInfos) != 1 {
			return nil, fmt.Errorf("multi rootfs mounted")
		}
		info := mountInfos[0]
		return buildinstruction.NewMountTarget(info.Target, info.Upper, info.Lowers)
	}

	lowerLayers := buildinstruction.GetBaseLayersPath(baseLayers)
	for _, info := range mountInfos {
		// if info.Lowers equal lowerLayers,means image already mounted.
		if strings.Join(lowerLayers, ":") == strings.Join(info.Lowers, ":") {
			logger.Info("get mount dir :%s success ", info.Target)
			//nolint
			return buildinstruction.NewMountTarget(info.Target, info.Upper, info.Lowers)
		}
	}

	return mountRootfs(lowerLayers)
}

func mountRootfs(res []string) (*buildinstruction.MountTarget, error) {
	mounter, err := buildinstruction.NewMountTarget("", "", res)
	if err != nil {
		return nil, err
	}

	err = mounter.TempMount()
	if err != nil {
		return nil, err
	}
	return mounter, nil
}
