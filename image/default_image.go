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
	"strings"

	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerioutils "github.com/docker/docker/pkg/ioutils"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerprogress "github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/distributionutil"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// DefaultImageService is the default service, which is used for image pull/push
type DefaultImageService struct {
	ForceDeleteImage bool // sealer rmi -f
	imageStore       store.ImageStore
}

// PullIfNotExist is used to pull image if not exists locally
func (d DefaultImageService) PullIfNotExist(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	_, err = d.imageStore.GetByName(named.Raw())
	if err == nil {
		logger.Info("image %s already exists", named.Raw())
		return nil
	}

	return d.Pull(imageName)
}

// PullIfNotExistAndReturnImage is used to pull image if not exists locally and return Image
func (d DefaultImageService) PullIfNotExistAndReturnImage(imageName string) (*v1.Image, error) {
	var image *v1.Image
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return nil, err
	}
GetImageByName:
	image, err = d.imageStore.GetByName(named.Raw())
	if err == nil {
		return image, nil
	}
	err = d.Pull(imageName)
	if err != nil {
		return nil, err
	}
	goto GetImageByName
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
	err = d.imageStore.Save(*image, named.Raw())
	if err == nil {
		dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Success to Pull Image %s", named.Raw()))
	}
	return err
}

// Push push local image to remote registry
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

// Login login into a registry, for saving auth info in ~/.docker/config.json
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

func (d DefaultImageService) Delete(imageName string) error {
	var (
		images        []*v1.Image
		image         *v1.Image
		imageTagCount int
		imageID       string
		imageStore    = d.imageStore
	)
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	imageMetadataMap, err := imageStore.GetImageMetadataMap()
	if err != nil {
		return err
	}
	// example ImageName : 7e2e51b85680d827fae08853dea32ad6:latest
	// example ImageID :   7e2e51b85680d827fae08853dea32ad6
	// https://github.com/alibaba/sealer/blob/f9d609c7fede47a7ac229bcd03d92dd0429b5038/image/reference/util.go#L59
	imageMetadata, ok := imageMetadataMap[named.Raw()]
	if !ok && strings.Contains(imageName, ":") {
		return fmt.Errorf("failed to find image with name %s", imageName)
	}

	if strings.Contains(imageName, ":") {
		//1.untag image
		if err = imageStore.DeleteByName(imageName); err != nil {
			return fmt.Errorf("failed to untag image %s, err: %w", imageName, err)
		}
		image, err = imageStore.GetByID(imageMetadata.ID)
		imageID = imageMetadata.ID
	} else {
		if err = imageStore.DeleteByID(imageName, d.ForceDeleteImage); err != nil {
			return err
		}
		image, err = imageStore.GetByID(imageName)
		imageID = imageName
	}

	if err != nil {
		return fmt.Errorf("failed to get image metadata for image %s, err: %w", imageName, err)
	}
	logger.Info("untag image %s succeeded", imageName)

	for _, value := range imageMetadataMap {
		tmpImage, err := imageStore.GetByID(value.ID)
		if err != nil {
			continue
		}
		if value.ID == imageID {
			imageTagCount++
			if imageTagCount > 1 {
				continue
			}
		}
		images = append(images, tmpImage)
	}
	if imageTagCount != 1 && !d.ForceDeleteImage {
		return nil
	}

	err = store.DeleteImageLocal(image.Spec.ID)
	if err != nil {
		return err
	}

	layer2ImageNames := layer2ImageMap(images)
	// TODO: find a atomic way to delete layers and image
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return err
	}

	for _, layer := range image.Spec.Layers {
		layerID := store.LayerID(layer.ID)
		if isLayerDeletable(layer2ImageNames, layerID) {
			err = layerStore.Delete(layerID)
			if err != nil {
				// print log and continue to delete other layers of the image
				logger.Error("Fail to delete image %s's layer %s", imageName, layerID)
			}
		}
	}

	logger.Info("image %s delete success", imageName)
	return nil
}

func isLayerDeletable(layer2ImageNames map[store.LayerID][]string, layerID store.LayerID) bool {
	return len(layer2ImageNames[layerID]) <= 1
}

// layer2ImageMap accepts a directory parameter which contains image metadata.
// It reads these metadata and saves the layer and image relationship in a map.
func layer2ImageMap(images []*v1.Image) map[store.LayerID][]string {
	var layer2ImageNames = make(map[store.LayerID][]string)
	for _, image := range images {
		for _, layer := range image.Spec.Layers {
			layerID := store.LayerID(layer.ID)
			layer2ImageNames[layerID] = append(layer2ImageNames[layerID], image.Spec.ID)
		}
	}
	return layer2ImageNames
}
