// Copyright © 2022 Alibaba Group Holding Ltd.
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

package imagedistributor

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/common"
	imagecommon "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/os/fs"
)

type buildAhMounter struct {
	imageEngine imageengine.Interface
}

func (b buildAhMounter) Mount(imageName string, platform v1.Platform) (string, error) {
	path := platform.OS + "_" + platform.Architecture + "_" + platform.Variant
	mountDir := filepath.Join(common.DefaultSealerDataDir, path)
	if err := b.imageEngine.Pull(&imagecommon.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   platform.ToString(),
	}); err != nil {
		return "", err
	}

	if _, err := b.imageEngine.CreateWorkingContainer(&imagecommon.BuildRootfsOptions{
		DestDir:       mountDir,
		ImageNameOrID: imageName,
	}); err != nil {
		return "", err
	}
	return mountDir, nil
}

func (b buildAhMounter) Umount(mountDir string) error {
	if err := b.imageEngine.RemoveContainer(&imagecommon.RemoveContainerOptions{
		ContainerNamesOrIDs: nil,
		All:                 true,
	}); err != nil {
		return fmt.Errorf("failed to remove mounted dir %s: %v", mountDir, err)
	}

	if err := fs.FS.RemoveAll(mountDir); err != nil {
		return err
	}
	return nil
}

func NewBuildAhMounter(imageEngine imageengine.Interface) Mounter {
	return buildAhMounter{
		imageEngine: imageEngine,
	}
}

type ImagerMounter struct {
	Mounter
	hostsPlatform map[v1.Platform][]net.IP
}

func (c ImagerMounter) Mount(imageName string) (map[string]string, error) {
	// map[ip.string][mountDir]
	imageMountInfo := make(map[string]string)

	for platform, hosts := range c.hostsPlatform {
		mountDir, err := c.Mounter.Mount(imageName, platform)
		if err != nil {
			return nil, fmt.Errorf("failed to mount image with platform %s:%v", platform.ToString(), err)
		}

		for _, host := range hosts {
			imageMountInfo[host.String()] = mountDir
		}
	}

	return imageMountInfo, nil
}

func (c ImagerMounter) Umount(imageMountInfo map[string]string) error {
	for _, mountDir := range imageMountInfo {
		err := c.Mounter.Umount(mountDir)
		if err != nil {
			return fmt.Errorf("failed to umount %s:%v", mountDir, err)
		}
	}
	return nil
}

func NewImageMounter(hostsPlatform map[v1.Platform][]net.IP) (*ImagerMounter, error) {
	imageEngine, err := imageengine.NewImageEngine(imagecommon.EngineGlobalConfigurations{})
	if err != nil {
		return nil, err
	}
	c := &ImagerMounter{
		hostsPlatform: hostsPlatform,
	}
	c.Mounter = NewBuildAhMounter(imageEngine)
	return c, nil
}
