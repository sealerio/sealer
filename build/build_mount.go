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

package build

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/runtime"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

type MountTarget struct {
	driver     mount.Interface
	TempTarget string
	TempUpper  string
	LowLayers  []string
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
	utils.CleanDirs(m.TempUpper)
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
func NewMountTarget(target, upper string, LowLayers []string) (*MountTarget, error) {
	if len(LowLayers) == 0 {
		return nil, fmt.Errorf("mount lowlayers can not be nil")
	}
	if target == "" {
		tmp, err := utils.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create target %s:%v", tmp, err)
		}
		target = tmp
	}
	if upper == "" {
		tmp, err := utils.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create upper %s:%v", tmp, err)
		}
		upper = tmp
	}
	return &MountTarget{
		driver:     mount.NewMountDriver(),
		TempTarget: target,
		TempUpper:  upper,
		LowLayers:  LowLayers,
	}, nil
}

func NewRegistryCache() (*MountTarget, error) {
	//$rootfs/registry
	dir := GetRegistryBindDir()
	if dir == "" {
		return nil, nil
	}
	rootfs := filepath.Dir(dir)
	// if rootfs dir not mounted, unable to get cache image layer. need to mount rootfs before init-registry
	mount, upper := mount.GetMountDetails(rootfs)
	if !mount {
		mountTarget, err := NewMountTarget(rootfs, runtime.RegistryMountUpper, []string{rootfs})
		if err != nil {
			return nil, err
		}
		str, err := utils.RunSimpleCmd(fmt.Sprintf("rm -rf %s && mkdir -p %s", runtime.RegistryMountUpper, runtime.RegistryMountUpper))
		if err != nil {
			logger.Error(str)
			return nil, err
		}
		err = mountTarget.TempMount()
		if err != nil {
			return nil, fmt.Errorf("failed to mount %s, %v", rootfs, err)
		}
		str, err = utils.RunSimpleCmd(fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", rootfs, rootfs))
		logger.Info(str)
		if err != nil {
			return nil, fmt.Errorf("failed to init registry, %s", err)
		}
		return mountTarget, nil
	}

	logger.Info("get registry cache dir :%s success ", dir)
	return NewMountTarget(rootfs, upper, []string{rootfs})
}
