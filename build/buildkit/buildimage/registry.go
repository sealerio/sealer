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
	"path/filepath"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"
	"github.com/alibaba/sealer/client/docker"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

func getRegistryBindDir() string {
	// check is docker running runtime.RegistryName
	// check bind dir
	var registryName = runtime.RegistryName
	var registryDest = runtime.RegistryBindDest

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return ""
	}

	containers, err := dockerClient.GetContainerListByName(registryName)

	if err != nil {
		return ""
	}

	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == "bind" && m.Destination == registryDest {
				return m.Source
			}
		}
	}

	return ""
}

func NewRegistryCache(baseLayers []v1.Layer) (*buildinstruction.MountTarget, error) {
	//$rootfs/registry
	dir := getRegistryBindDir()
	if dir == "" {
		return mountRootfs(buildinstruction.GetBaseLayersPath(baseLayers))
	}
	rootfs := filepath.Dir(dir)
	// if already mounted ,read mount details set to RootfsMountTarget and return.
	// Negative examples:
	//if pull images failed or exec kubefile instruction failed, rerun build again,will cache part images.
	isMounted, info := mount.GetMountDetails(rootfs)
	if isMounted {
		logger.Info("get registry cache dir :%s success ", dir)
		//nolint
		return buildinstruction.NewMountTarget(rootfs, info.Upper, utils.Reverse(info.Lowers))
	}

	return nil, fmt.Errorf("sealer registry is already exist,but not mounted")
}

func mountRootfs(res []string) (*buildinstruction.MountTarget, error) {
	rootfs, err := buildinstruction.NewMountTarget("", "", res)
	if err != nil {
		return nil, err
	}

	err = rootfs.TempMount()
	if err != nil {
		return nil, err
	}
	return rootfs, nil
}
