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
	"os"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/os/fs"
)

type buildAhMounter struct {
	imageEngine imageengine.Interface
}

func (b buildAhMounter) Mount(imageName string, platform v1.Platform, dest string) (string, string, string, error) {
	mountDir := filepath.Join(dest,
		strings.ReplaceAll(imageName, "/", "_"),
		strings.Join([]string{platform.OS, platform.Architecture, platform.Variant}, "_"))

	imageID, err := b.imageEngine.Pull(&options.PullOptions{
		Quiet:      false,
		PullPolicy: "missing",
		Image:      imageName,
		Platform:   platform.ToString(),
	})
	if err != nil {
		return "", "", "", err
	}

	if err := fs.FS.MkdirAll(filepath.Dir(mountDir)); err != nil {
		return "", "", "", err
	}

	id, err := b.imageEngine.CreateWorkingContainer(&options.BuildRootfsOptions{
		DestDir:       mountDir,
		ImageNameOrID: imageID,
	})

	if err != nil {
		return "", "", "", err
	}
	return mountDir, id, imageID, nil
}

func (b buildAhMounter) Umount(mountDir, cid string) error {
	if err := fs.FS.RemoveAll(mountDir); err != nil {
		return fmt.Errorf("failed to remove mount dir %s: %v", mountDir, err)
	}

	if err := b.imageEngine.RemoveContainer(&options.RemoveContainerOptions{
		ContainerNamesOrIDs: []string{cid},
	}); err != nil {
		return fmt.Errorf("failed to remove working container: %v", err)
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
	rootDir       string
	hostsPlatform map[v1.Platform][]net.IP
}

type ClusterImageMountInfo struct {
	// target hosts ip list, not all cluster ips.
	Hosts       []net.IP
	Platform    v1.Platform
	MountDir    string
	ContainerID string
	ImageID     string
}

func (c ImagerMounter) Mount(imageName string) ([]ClusterImageMountInfo, error) {
	var imageMountInfos []ClusterImageMountInfo
	for platform, hosts := range c.hostsPlatform {
		mountDir, cid, imageID, err := c.Mounter.Mount(imageName, platform, c.rootDir)
		if err != nil {
			return nil, fmt.Errorf("failed to mount image %s with platform %s:%v", imageName, platform.ToString(), err)
		}
		imageMountInfos = append(imageMountInfos, ClusterImageMountInfo{
			Hosts:       hosts,
			Platform:    platform,
			MountDir:    mountDir,
			ContainerID: cid,
			ImageID:     imageID,
		})
	}

	return imageMountInfos, nil
}

func (c ImagerMounter) Umount(imageName string, imageMountInfo []ClusterImageMountInfo) error {
	for _, info := range imageMountInfo {
		err := c.Mounter.Umount(info.MountDir, info.ContainerID)
		if err != nil {
			return fmt.Errorf("failed to umount %s:%v", info.MountDir, err)
		}
	}

	// delete all mounted images
	if err := fs.FS.RemoveAll(c.rootDir); err != nil {
		return err
	}
	return nil
}

func NewImageMounter(imageEngine imageengine.Interface, hostsPlatform map[v1.Platform][]net.IP) (*ImagerMounter, error) {
	tempDir, err := os.MkdirTemp("", "sealer-mount-tmp")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp mount dir, err: %v", err)
	}

	c := &ImagerMounter{
		// todo : user could set this value by env or sealer config
		rootDir:       tempDir,
		hostsPlatform: hostsPlatform,
	}

	c.Mounter = NewBuildAhMounter(imageEngine)
	return c, nil
}
