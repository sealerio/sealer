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

package image

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerioutils "github.com/docker/docker/pkg/ioutils"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerprogress "github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/distributionutil"
	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// DefaultImageService is the default service, which is used for image pull/push
type DefaultImageService struct {
	imageStore store.ImageStore
}

// PullIfNotExist is used to pull image if not exists locally
func (d DefaultImageService) PullIfNotExist(imageName string, platform *v1.Platform) error {
	img, err := d.GetImageByName(imageName, platform)
	if err != nil {
		return err
	}
	if img != nil {
		logger.Debug("image %s already exists", imageName)
		return nil
	}

	return d.Pull(imageName)
}

func (d DefaultImageService) GetImageByName(imageName string, platform *v1.Platform) (*v1.Image, error) {
	var img *v1.Image
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return nil, err
	}
	img, err = d.imageStore.GetByName(named.Raw(), platform)
	if err == nil {
		logger.Debug("image %s already exists", named)
		return img, nil
	}
	return nil, nil
}

// Pull always do pull action
func (d DefaultImageService) Pull(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}
	var (
		reader, writer  = io.Pipe()
		writeFlusher    = dockerioutils.NewWriteFlusher(writer)
		progressChanOut = streamformatter.NewJSONProgressOutput(writeFlusher, false)
		streamOut       = dockerstreams.NewOut(common.StdOut)
	)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
	}()

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return err
	}

	puller, err := distributionutil.NewPuller(named, distributionutil.Config{
		LayerStore:     layerStore,
		ProgressOutput: progressChanOut,
	})
	if err != nil {
		return err
	}

	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, streamOut, nil)
		if err != nil && err != io.ErrClosedPipe {
			logger.Warn("error occurs in display progressing, err: %s", err)
		}
	}()

	dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Start to Pull Image %s", named.Raw()))
	image, err := puller.Pull(context.Background(), named)
	if err != nil {
		return err
	}
	// TODO use image store to do the job next
	err = d.imageStore.Save(*image)
	if err == nil {
		dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Success to Pull Image %s", named.Raw()))
	}
	return err
}

// Push local image to remote registry
func (d DefaultImageService) Push(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}
	var (
		reader, writer  = io.Pipe()
		writeFlusher    = dockerioutils.NewWriteFlusher(writer)
		progressChanOut = streamformatter.NewJSONProgressOutput(writeFlusher, false)
		streamOut       = dockerstreams.NewOut(common.StdOut)
	)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
	}()

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return err
	}

	pusher, err := distributionutil.NewPusher(named,
		distributionutil.Config{
			LayerStore:     layerStore,
			ProgressOutput: progressChanOut,
		})
	if err != nil {
		return err
	}
	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, streamOut, nil)
		// reader may be closed in another goroutine
		// so do not log warn when err == io.ErrClosedPipe
		if err != nil && err != io.ErrClosedPipe {
			logger.Warn("error occurs in display progressing, err: %s", err)
		}
	}()

	dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Start to Push Image %s", named.Raw()))
	err = pusher.Push(context.Background(), named)
	if err == nil {
		dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Success to Push Image %s", named.CompleteName()))
	}
	return err
}

// Login into a registry, for saving auth info in ~/.docker/config.json
func (d DefaultImageService) Login(RegistryURL, RegistryUsername, RegistryPasswd string) error {
	err := distributionutil.Login(context.Background(), &types.AuthConfig{ServerAddress: RegistryURL, Username: RegistryUsername, Password: RegistryPasswd})
	if err != nil {
		return fmt.Errorf("failed to authenticate %s: %v", RegistryURL, err)
	}
	if err := utils.SetDockerConfig(RegistryURL, RegistryUsername, RegistryPasswd); err != nil {
		return err
	}
	logger.Info("%s login %s success", RegistryUsername, RegistryURL)
	return nil
}

// Delete image layer, image meta, and local registry.
// if --force=true, will delete all image data.
// if platforms not nil will delete all the related platform images.
func (d DefaultImageService) Delete(imageName string, force bool, platforms []*v1.Platform) error {
	var (
		err               error
		named             reference.Named
		imageStore        = d.imageStore
		deleteImageIDList []string
		latestImageIDList []string
	)

	named, err = reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	if force {
		imageMetadataMap, err := imageStore.GetImageMetadataMap()
		if err != nil {
			return err
		}
		manifestList, ok := imageMetadataMap[named.Raw()]
		if !ok {
			return fmt.Errorf("image %s not found", imageName)
		}
		for _, m := range manifestList.Manifests {
			deleteImageIDList = append(deleteImageIDList, m.ID)
		}
		if err = imageStore.DeleteByName(named.Raw(), nil); err != nil {
			return fmt.Errorf("failed to delete image %s, err: %v", imageName, err)
		}
	}

	for _, plat := range platforms {
		img, err := imageStore.GetByName(named.Raw(), plat)
		if err != nil {
			return fmt.Errorf("image %s not found %v", named.Raw(), err)
		}
		deleteImageIDList = append(deleteImageIDList, img.Spec.ID)
		if err = imageStore.DeleteByName(named.Raw(), plat); err != nil {
			return fmt.Errorf("failed to delete image %s, err: %v", imageName, err)
		}
	}

	deleteImageIDList = utils.RemoveDuplicate(deleteImageIDList)
	imageMetadataMap, err := imageStore.GetImageMetadataMap()
	if err != nil {
		return err
	}

	for _, imageMetadata := range imageMetadataMap {
		for _, m := range imageMetadata.Manifests {
			latestImageIDList = append(latestImageIDList, m.ID)
		}
	}
	// delete image.yaml file which id not in current imageMetadataMap.
	for _, id := range deleteImageIDList {
		if utils.InList(id, latestImageIDList) {
			continue
		}
		err = store.DeleteImageLocal(id)
		if err != nil {
			return err
		}
	}

	logger.Info("image %s delete success", imageName)
	return d.Prune()
}

// Prune delete the unused Layer in the `DefaultLayerDir` directory
func (d DefaultImageService) Prune() error {
	imageMetadataMap, err := d.imageStore.GetImageMetadataMap()
	var allImageLayerDirs []string
	if err != nil {
		return err
	}

	for _, imageMetadata := range imageMetadataMap {
		for _, m := range imageMetadata.Manifests {
			image, err := d.imageStore.GetByID(m.ID)
			if err != nil {
				return err
			}
			res, err := GetImageLayerDirs(image)
			if err != nil {
				return err
			}
			allImageLayerDirs = append(allImageLayerDirs, res...)
		}
	}

	allImageLayerDirs = utils.RemoveDuplicate(allImageLayerDirs)
	dirs, err := store.GetDirListInDir(common.DefaultLayerDir)
	if err != nil {
		return err
	}
	dirs = utils.RemoveStrSlice(dirs, allImageLayerDirs)
	for _, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		_, err = common.StdOut.WriteString(fmt.Sprintf("%s layer deleted\n", filepath.Base(dir)))
		if err != nil {
			return err
		}
	}
	return nil
}
