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

package buildinstruction

import (
	"fmt"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

type MountTarget struct {
	driver     mount.Interface
	TempTarget string
	TempUpper  string
	//driver.Mount will reverse lowers,so here we keep the order same with the image layer.
	LowLayers []string
}

func (m MountTarget) TempMount() error {
	err := m.driver.Mount(m.TempTarget, m.TempUpper, m.LowLayers...)
	if err != nil {
		return fmt.Errorf("failed to mount target %s:%v", m.TempTarget, err)
	}
	return nil
}

func (m MountTarget) TempUMount() error {
	err := m.driver.Unmount(m.TempTarget)
	if err != nil {
		return fmt.Errorf("failed to mount target %s:%v", m.TempTarget, err)
	}
	return nil
}

func (m MountTarget) CleanUp() {
	if err := m.driver.Unmount(m.TempTarget); err != nil {
		logger.Warn(fmt.Errorf("failed to umount %s:%v", m.TempTarget, err))
	}

	utils.CleanDirs(m.TempUpper, m.TempTarget)
}

func (m MountTarget) GetMountUpper() string {
	return m.TempUpper
}

func (m MountTarget) GetLowLayers() []string {
	return m.LowLayers
}

func (m MountTarget) GetMountTarget() string {
	return m.TempTarget
}

//NewMountTarget will create temp dir if target or upper is nil. it is convenient for use in build stage
func NewMountTarget(target, upper string, lowLayers []string) (*MountTarget, error) {
	if len(lowLayers) == 0 {
		tmp, err := utils.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp lower %s:%v", tmp, err)
		}
		lowLayers = append(lowLayers, tmp)
	}
	if target == "" {
		tmp, err := utils.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp target %s:%v", tmp, err)
		}
		target = tmp
	}
	if upper == "" {
		tmp, err := utils.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp upper %s:%v", tmp, err)
		}
		upper = tmp
	}
	return &MountTarget{
		driver:     mount.NewMountDriver(),
		TempTarget: target,
		TempUpper:  upper,
		LowLayers:  lowLayers,
	}, nil
}
