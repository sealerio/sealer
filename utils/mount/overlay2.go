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

// +build linux

package mount

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	// merged layer files
	Mount(target string, upperDir string, layers ...string) error
	Unmount(target string) error
}

type Overlay2 struct {
}

func NewMountDriver() Interface {
	if supportsOverlay() {
		return &Overlay2{}
	}
	return &Default{}
}

func supportsOverlay() bool {
	if err := exec.Command("modprobe", "overlay").Run(); err != nil {
		return false
	}
	f, err := os.Open("/proc/filesystems")
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if s.Text() == "nodev\toverlay" {
			return true
		}
	}
	return false
}

// using overlay2 to merged layer files
func (o *Overlay2) Mount(target string, upperLayer string, layers ...string) error {
	if target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if len(layers) == 0 {
		return fmt.Errorf("layers cannot be empty")
	}
	workdir := path.Join(target, "work")
	if err := utils.Mkdir(workdir); err != nil {
		return fmt.Errorf("create workdir failed")
	}
	var err error
	defer func() {
		if err != nil {
			_ = os.RemoveAll(workdir)
		}
	}()
	mountData := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", strings.Join(utils.Reverse(layers), ":"), upperLayer, workdir)
	logger.Debug("mount data : %s", mountData)
	if err = mount("overlay", target, "overlay", 0, mountData); err != nil {
		return fmt.Errorf("error creating overlay mount to %s: %v", target, err)
	}
	return nil
}

// Unmount target
func (o *Overlay2) Unmount(target string) error {
	return unmount(target, 0)
}

func mount(device, target, mType string, flag uintptr, data string) error {
	if err := syscall.Mount(device, target, mType, flag, data); err != nil {
		return err
	}

	// If we have a bind mount or remount, remount...
	if flag&syscall.MS_BIND == syscall.MS_BIND && flag&syscall.MS_RDONLY == syscall.MS_RDONLY {
		return syscall.Mount(device, target, mType, flag|syscall.MS_REMOUNT, data)
	}
	return nil
}

func unmount(target string, flag int) error {
	return syscall.Unmount(target, flag)
}

type Info struct {
	Target string
	Upper  string
	Lowers []string
}

func GetMountDetails(target string) (bool, *Info) {
	cmd := fmt.Sprintf("mount | grep %s", target)
	result, err := utils.RunSimpleCmd(cmd)
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
		Lowers: lowers,
	}
}
