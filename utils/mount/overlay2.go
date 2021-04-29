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

	"gitlab.alibaba-inc.com/seadent/pkg/utils"
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
	mountData := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", strings.Join(layers, ":"), upperLayer, workdir)
	if err = mount("overlay", target, "overlay", 0, mountData); err != nil {
		//if mount failed, to unmount
		if err = unmount(target, 0); err != nil {
			return err
		}
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
