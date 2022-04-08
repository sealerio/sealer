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

//go:build linux
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
)

type Interface interface {
	// Mount merged layer files
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
	defer func() {
		if err := f.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if s.Text() == "nodev\toverlay" {
			return true
		}
	}
	return false
}

// Mount using overlay2 to merged layer files
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

	var indexOff string
	// figure out whether "index=off" option is recognized by the kernel
	_, err = os.Stat("/sys/module/overlay/parameters/index")
	switch {
	case err == nil:
		indexOff = "index=off,"
	case os.IsNotExist(err):
		// old kernel, no index -- do nothing
	default:
		logger.Warn("Unable to detect whether overlay kernel module supports index parameter: %s", err)
	}

	mountData := fmt.Sprintf("%slowerdir=%s,upperdir=%s,workdir=%s", indexOff, strings.Join(utils.Reverse(layers), ":"), upperLayer, workdir)
	logger.Debug("mount data : %s", mountData)
	if err = mount("overlay", target, "overlay", 0, mountData); err != nil {
		return fmt.Errorf("error creating overlay mount to %s: %v", target, err)
	}
	return nil
}

// Unmount target
func (o *Overlay2) Unmount(target string) error {
	return unmount(target, syscall.MNT_FORCE)
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
