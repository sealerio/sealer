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

package mount

import (
	"fmt"
	"strings"

	"github.com/sealerio/sealer/utils/os/fs"

	"github.com/sealerio/sealer/utils/exec"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/shirou/gopsutil/disk"

	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/utils/ssh"
)

type Service interface {
	TempMount() error
	TempUMount() error
	CleanUp()
	GetMountTarget() string
	GetMountUpper() string
	GetLowLayers() []string
}

type mounter struct {
	driver     Interface
	fs         fs.Interface
	TempTarget string
	TempUpper  string
	//driver.Mount will reverse lowers,so here we keep the order same with the image layer.
	LowLayers []string
}

func (m mounter) TempMount() error {
	err := m.driver.Mount(m.TempTarget, m.TempUpper, m.LowLayers...)
	if err != nil {
		return fmt.Errorf("failed to mount target %s:%v", m.TempTarget, err)
	}
	return nil
}

func (m mounter) TempUMount() error {
	err := m.driver.Unmount(m.TempTarget)
	if err != nil {
		return fmt.Errorf("failed to umount target %s:%v", m.TempTarget, err)
	}
	return nil
}

func (m mounter) CleanUp() {
	var err error

	err = m.driver.Unmount(m.TempTarget)
	if err != nil {
		logger.Warn("failed to umount %s:%v", m.TempTarget, err)
	}

	err = m.fs.RemoveAll(m.TempUpper, m.TempTarget)
	if err != nil {
		logger.Warn("failed to delete %s,%s", m.TempUpper, m.TempTarget)
	}
}

func (m mounter) GetMountUpper() string {
	return m.TempUpper
}

func (m mounter) GetLowLayers() []string {
	return m.LowLayers
}

func (m mounter) GetMountTarget() string {
	return m.TempTarget
}

//NewMountService will create temp dir if target or upper is nil. it is convenient for use in build stage
func NewMountService(target, upper string, lowLayers []string) (Service, error) {
	f := fs.NewFilesystem()
	if len(lowLayers) == 0 {
		tmp, err := f.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp lower %s:%v", tmp, err)
		}
		lowLayers = append(lowLayers, tmp)
	}
	if target == "" {
		tmp, err := f.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp target %s:%v", tmp, err)
		}
		target = tmp
	}
	if upper == "" {
		tmp, err := f.MkTmpdir()
		if err != nil {
			return nil, fmt.Errorf("failed to create tmp upper %s:%v", tmp, err)
		}
		upper = tmp
	}
	return &mounter{
		fs:         f,
		driver:     NewMountDriver(),
		TempTarget: target,
		TempUpper:  upper,
		LowLayers:  lowLayers,
	}, nil
}

//NewMountServiceByTarget will filter file system by target,if not existed,return false.
func NewMountServiceByTarget(target string) Service {
	mounted, info := GetMountDetails(target)
	if !mounted {
		return nil
	}
	return &mounter{
		driver:     NewMountDriver(),
		TempTarget: target,
		TempUpper:  info.Upper,
		LowLayers:  info.Lowers,
	}
}

type Info struct {
	Target string
	Upper  string
	Lowers []string
}

func GetMountDetails(target string) (bool, *Info) {
	cmd := fmt.Sprintf("mount | grep %s", target)
	result, err := exec.RunSimpleCmd(cmd)
	if err != nil {
		return false, nil
	}
	return mountCmdResultSplit(result, target)
}

func GetRemoteMountDetails(s ssh.Interface, ip string, target string) (bool, *Info) {
	result, err := s.Cmd(ip, fmt.Sprintf("mount | grep %s", target))
	if err != nil {
		return false, nil
	}
	return mountCmdResultSplit(string(result), target)
}

func mountCmdResultSplit(result string, target string) (bool, *Info) {
	if !strings.Contains(result, target) {
		return false, nil
	}

	data := strings.Split(result, ",upperdir=")
	if len(data) < 2 {
		return false, nil
	}

	lowers := strings.Split(strings.Split(data[0], ",lowerdir=")[1], ":")
	upper := strings.TrimSpace(strings.Split(data[1], ",workdir=")[0])
	return true, &Info{
		Target: target,
		Upper:  upper,
		Lowers: strUtils.Reverse(lowers),
	}
}

func GetBuildMountInfo(filter string) []Info {
	var infos []Info
	var mp []string
	ps, _ := disk.Partitions(true)
	for _, p := range ps {
		if p.Fstype == "overlay" && strings.Contains(p.Mountpoint, "sealer") &&
			strings.Contains(p.Mountpoint, filter) {
			mp = append(mp, p.Mountpoint)
		}
	}
	for _, p := range mp {
		_, info := GetMountDetails(p)
		if info != nil {
			infos = append(infos, *info)
		}
	}
	return infos
}
