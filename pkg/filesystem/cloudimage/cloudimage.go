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

package cloudimage

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	"github.com/sealerio/sealer/utils/os/fs"

	osi "github.com/sealerio/sealer/utils/os"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/store"
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
	imageStore store.ImageStore
	fs         fs.Interface
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
				return fmt.Errorf("failed to unmount dir %s,err: %v", filepath.Join(mountRootDir, f.Name()), err)
			}
		}
	}

	return m.fs.RemoveAll(mountRootDir)
}

func (m *mounter) mountImage(cluster *v2.Cluster) error {
	var (
		mountDirs = make(map[string]bool)
		driver    = mount.NewMountDriver()
		err       error
	)
	clusterPlatform, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}
	clusterPlatform["local"] = *platform.GetDefaultPlatform()
	for _, v := range clusterPlatform {
		pfm := v
		mountDir := platform.GetMountCloudImagePlatformDir(cluster.Name, pfm)
		upperDir := filepath.Join(mountDir, "upper")
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
				return fmt.Errorf("failed to clean %s, %v", mountDir, err)
			}
		}
		Image, err := m.imageStore.GetByName(cluster.Spec.Image, &pfm)
		if err != nil {
			return err
		}
		layers, err := image.GetImageLayerDirs(Image)
		if err != nil {
			return fmt.Errorf("get layers failed: %v", err)
		}

		if err = m.fs.MkdirAll(upperDir); err != nil {
			return fmt.Errorf("create upperdir failed, %s", err)
		}
		if err = driver.Mount(mountDir, upperDir, layers...); err != nil {
			return fmt.Errorf("mount files failed %v", err)
		}
		// use env list to render image mount dir: etc,charts,manifests.
		err = renderENV(mountDir, cluster.GetAllIPList(), env.NewEnvProcessor(cluster))
		if err != nil {
			return err
		}
	}
	return nil
}

func renderENV(imageMountDir string, ipList []string, p env.Interface) error {
	var (
		renderEtc       = filepath.Join(imageMountDir, common.EtcDir)
		renderChart     = filepath.Join(imageMountDir, common.RenderChartsDir)
		renderManifests = filepath.Join(imageMountDir, common.RenderManifestsDir)
	)

	for _, ip := range ipList {
		for _, dir := range []string{renderEtc, renderChart, renderManifests} {
			if osi.IsFileExist(dir) {
				err := p.RenderAll(ip, dir)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func NewCloudImageMounter(is store.ImageStore) (Interface, error) {
	return &mounter{
		imageStore: is,
		fs:         fs.NewFilesystem(),
	}, nil
}
