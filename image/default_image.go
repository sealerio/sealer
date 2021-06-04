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
	"strings"

	"github.com/alibaba/sealer/image/distributionutil"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	imageutils "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/registry"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerutils "github.com/docker/docker/distribution/utils"
	dockerioutils "github.com/docker/docker/pkg/ioutils"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerprogress "github.com/docker/docker/pkg/progress"
)

// DefaultImageService is the default service, which is used for image pull/push
type DefaultImageService struct {
	ForceDeleteImage bool // sealer rmi -f
}

// PullIfNotExist is used to pull image if not exists locally
func (d DefaultImageService) PullIfNotExist(imageName string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	_, err = imageutils.GetImage(named.Raw())
	if err == nil {
		logger.Info("image %s already exists", named.Raw())
		return nil
	}

	return d.Pull(imageName)
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
		progressChan    = make(chan dockerprogress.Progress, 100)
		progressChanOut = dockerprogress.ChanOutput(progressChan)
		streamOut       = dockerstreams.NewOut(os.Stdout)
	)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
		close(progressChan)
	}()
	go func() {
		dockerutils.WriteDistributionProgress(func() {}, writeFlusher, progressChan)
	}()

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return err
	}

	authInfo, err := utils.GetDockerAuthInfoFromDocker(named.Domain())
	if err != nil {
		logger.Warn("failed to get auth info, err: %s", err)
	}

	puller, err := distributionutil.NewPuller(distributionutil.Config{
		LayerStore:     layerStore,
		ProgressOutput: progressChanOut,
		AuthInfo:       authInfo,
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
	return store.SyncImageLocal(*image, named)
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
		progressChan    = make(chan dockerprogress.Progress, 100)
		progressChanOut = dockerprogress.ChanOutput(progressChan)
		streamOut       = dockerstreams.NewOut(os.Stdout)
	)
	defer func() {
		_ = reader.Close()
		_ = writer.Close()
		_ = writeFlusher.Close()
		close(progressChan)
	}()

	go func() {
		dockerutils.WriteDistributionProgress(func() {}, writeFlusher, progressChan)
	}()

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return err
	}

	authInfo, err := utils.GetDockerAuthInfoFromDocker(named.Domain())
	if err != nil {
		logger.Warn("failed to get docker info, err: %s", err)
	}

	pusher, err := distributionutil.NewPusher(distributionutil.Config{
		LayerStore:     layerStore,
		ProgressOutput: progressChanOut,
		AuthInfo:       authInfo,
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
	return pusher.Push(context.Background(), named)
}

// Login login into a registry, for saving auth info in ~/.docker/config.json
func (d DefaultImageService) Login(RegistryURL, RegistryUsername, RegistryPasswd string) error {
	_, err := registry.New(context.Background(), types.AuthConfig{ServerAddress: RegistryURL, Username: RegistryUsername, Password: RegistryPasswd}, registry.Opt{Insecure: true, Debug: true})
	if err != nil {
		logger.Error("%v authentication failed", RegistryURL)
		return err
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
	)
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	imageMetadataMap, err := imageutils.GetImageMetadataMap()
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
		if err = imageutils.DeleteImage(imageName); err != nil {
			return fmt.Errorf("failed to untag image %s, err: %w", imageName, err)
		}
		image, err = imageutils.GetImageByID(imageMetadata.ID)

		imageID = imageMetadata.ID
	} else {
		if err = imageutils.DeleteImageByID(imageName, d.ForceDeleteImage); err != nil {
			return err
		}
		image, err = imageutils.GetImageByID(imageName)
		imageID = imageName
	}

	if err != nil {
		return fmt.Errorf("failed to get image metadata for image %s, err: %w", imageName, err)
	}
	logger.Info("untag image %s succeeded", imageName)

	for _, value := range imageMetadataMap {
		tmpImage, err := imageutils.GetImageByID(value.ID)
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
		layerID := store.LayerID(layer.Hash)
		if isLayerDeletable(layer2ImageNames, layerID) || d.ForceDeleteImage {
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
			layerID := store.LayerID(layer.Hash)
			layer2ImageNames[layerID] = append(layer2ImageNames[layerID], image.Name)
		}
	}
	return layer2ImageNames
}
