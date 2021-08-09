package build

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
	utils.CleanDirs(m.TempTarget, m.TempUpper)
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

//func NewRegistryCache() (*MountTarget, error) {
//	dir := GetRegistryBindDir()
//	if dir == "" {
//		return nil, nil
//	}
//	// if registry dir not mounted, return
//	mounted, upper := GetMountDetails(dir)
//	if !mounted {
//		return nil, nil
//	}
//
//	logger.Info("get registry cache dir :%s success ", dir)
//	registryCache, err := NewMountTarget(dir,
//		upper, []string{dir})
//	if err != nil {
//		return nil, err
//	}
//
//	return registryCache, nil
//}
