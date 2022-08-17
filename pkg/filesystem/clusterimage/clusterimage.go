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

package clusterimage

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/sealerio/sealer/pkg/rootfs"

	"github.com/pkg/errors"

	options "github.com/sealerio/sealer/pkg/define/options"

	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/utils/os/fs"

	osi "github.com/sealerio/sealer/utils/os"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/env"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/mount"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
)

type Interface interface {
	MountImage(cluster *v2.Cluster) error
	UnMountImage(cluster *v2.Cluster) error
}

type mounter struct {
	imageEngine imageengine.Interface
	fs          fs.Interface
}

func (m *mounter) MountImage(cluster *v2.Cluster) error {
	return m.mountImage(cluster)
}

func (m *mounter) UnMountImage(cluster *v2.Cluster) error {
	return m.umountImage(cluster)
}

func (m *mounter) umountImage(cluster *v2.Cluster) error {
	mountRootDir := filepath.Join(common.DefaultClusterRootfsDir, cluster.Name, "mount")
	if !osi.IsFileExist(mountRootDir) {
		return nil
	}
	dir, err := ioutil.ReadDir(mountRootDir)
	if err != nil {
		return err
	}
	for _, f := range dir {
		if !f.IsDir() {
			continue
		}
		if isMount, _ := mount.GetMountDetails(filepath.Join(mountRootDir, f.Name())); isMount {
			err := utils.Retry(10, time.Second, func() error {
				return mount.NewMountDriver().Unmount(filepath.Join(mountRootDir, f.Name()))
			})
			if err != nil {
				return fmt.Errorf("failed to unmount dir(%s): %v", filepath.Join(mountRootDir, f.Name()), err)
			}
		}
	}

	// this will remove all buildah containers
	if err = m.imageEngine.RemoveContainer(&options.RemoveContainerOptions{
		ContainerNamesOrIDs: nil,
		All:                 true,
	}); err != nil {
		return fmt.Errorf("remove containers failed, you'd better execute a prune to remove it: %v", err)
	}

	return m.fs.RemoveAll(mountRootDir)
}

func (m *mounter) mountImage(cluster *v2.Cluster) error {
	var (
		mountDirs = make(map[string]bool)
		driver    = mount.NewMountDriver()
		image     = cluster.Spec.Image
		err       error
	)
	clusterPlatform, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}

	clusterPlatform["local"] = *platform.GetDefaultPlatform()
	for _, v := range clusterPlatform {
		pfm := v
		mountDir := platform.GetMountClusterImagePlatformDir(cluster.Name, pfm)
		if mountDirs[mountDir] {
			continue
		}
		mountDirs[mountDir] = true
		if isMount, _ := mount.GetMountDetails(mountDir); isMount {
			err = driver.Unmount(mountDir)
			if err != nil {
				return fmt.Errorf("%s already mount, and failed to umount %v", mountDir, err)
			}
		}
		if osi.IsFileExist(mountDir) {
			if err = m.fs.RemoveAll(mountDir); err != nil {
				return fmt.Errorf("failed to clean %s: %v", mountDir, err)
			}
		}

		if err = os.MkdirAll(filepath.Dir(mountDir), 0750); err != nil {
			return err
		}

		// build oci image rootfs under graphroot
		// and the rootfs will be linked to sealer rootfs.
		if _, err = m.imageEngine.CreateWorkingContainer(&options.BuildRootfsOptions{
			ImageNameOrID: image,
			DestDir:       mountDir,
		}); err != nil {
			return errors.Wrap(err, "failed to build rootfs when mounting image")
		}

		// use env list to render image mount dir: etc,charts,manifests.
		err = renderENV(mountDir, cluster.GetAllIPList(), env.NewEnvProcessor(cluster))
		if err != nil {
			return err
		}
	}
	return nil
}

func renderENV(imageMountDir string, ipList []net.IP, p env.Interface) error {
	var (
		renderEtc         = filepath.Join(imageMountDir, common.EtcDir)
		renderChart       = filepath.Join(imageMountDir, common.RenderChartsDir)
		renderManifests   = filepath.Join(imageMountDir, common.RenderManifestsDir)
		renderApplication = filepath.Join(imageMountDir, rootfs.GlobalManager.App().Root())
	)

	for _, ip := range ipList {
		for _, dir := range []string{renderEtc, renderChart, renderManifests, renderApplication} {
			if osi.IsFileExist(dir) {
				if err := p.RenderAll(ip, dir); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewClusterImageMounter(ie imageengine.Interface) (Interface, error) {
	return &mounter{
		imageEngine: ie,
		fs:          fs.NewFilesystem(),
	}, nil
}
