// Copyright © 2023 Alibaba Group Holding Ltd.
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
	"context"
	"fmt"
	"github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imageengine"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/os/fs"
	"github.com/sealerio/sealer/utils/weed"
	"path/filepath"
	"strings"
)

type weedMounter struct {
	imageEngine imageengine.Interface
	weedClient  weed.Deployer
}

func (w *weedMounter) Mount(imageName string, platform v1.Platform, dest string) (string, string, string, error) {
	mountDir := filepath.Join(dest,
		strings.ReplaceAll(imageName, "/", "_"),
		strings.Join([]string{platform.OS, platform.Architecture, platform.Variant}, "_"))

	imageID, err := w.imageEngine.Pull(&options.PullOptions{
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

	id, err := w.imageEngine.CreateWorkingContainer(&options.BuildRootfsOptions{
		DestDir:       mountDir,
		ImageNameOrID: imageID,
	})

	if err != nil {
		return "", "", "", err
	}

	// Upload the mounted files to the WeedFS cluster
	err = w.weedClient.UploadFile(context.Background(), mountDir)
	if err != nil {
		return "", "", "", err
	}

	return mountDir, id, imageID, nil
}

func (w *weedMounter) Umount(dir, containerID string) error {
	// Download the files from WeedFS cluster
	err := w.weedClient.DownloadFile(context.Background(), dir, dir)
	if err != nil {
		return err
	}

	// Umount the image and remove the working container
	err = w.imageEngine.RemoveContainer(&options.RemoveContainerOptions{
		ContainerNamesOrIDs: []string{containerID},
	})
	if err != nil {
		return err
	}

	// Remove the mounted files from the WeedFS cluster
	err = w.weedClient.RemoveFile(context.Background(), dir)
	if err != nil {
		return err
	}

	// Remove the local mount directory
	if err := fs.FS.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to remove mount dir %s: %v", dir, err)
	}

	return nil
}

func NewWeedMounter(imageEngine imageengine.Interface, config *weed.Config) Mounter {
	deployer := weed.NewDeployer(config)
	return &weedMounter{
		imageEngine: imageEngine,
		weedClient:  deployer,
	}
}
