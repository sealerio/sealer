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

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/schema2"
	dockerstreams "github.com/docker/cli/cli/streams"
	"github.com/docker/docker/api/types"
	dockerioutils "github.com/docker/docker/pkg/ioutils"
	dockerjsonmessage "github.com/docker/docker/pkg/jsonmessage"
	dockerprogress "github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/client/docker/auth"
	"github.com/sealerio/sealer/pkg/image/distributionutil"
	"github.com/sealerio/sealer/pkg/image/reference"
	"github.com/sealerio/sealer/pkg/image/store"
	v1 "github.com/sealerio/sealer/types/api/v1"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/strings"
)

// DefaultImageService is the default service, which is used for image pull/push
type DefaultImageService struct {
	imageStore store.ImageStore
}

// PullIfNotExist is used to pull image if not exists locally
func (d DefaultImageService) PullIfNotExist(imageName string, platforms []*v1.Platform) error {
	var plats []*v1.Platform
	for _, plat := range platforms {
		img, err := d.GetImageByName(imageName, plat)

		if err != nil {
			return err
		}
		// image not found
		if img == nil {
			plats = append(plats, plat)
		}
	}

	if len(plats) != 0 {
		return d.Pull(imageName, plats)
	}

	return nil
}

func (d DefaultImageService) GetImageByName(imageName string, platform *v1.Platform) (*v1.Image, error) {
	var img *v1.Image
	img, err := d.imageStore.GetByName(imageName, platform)
	if err == nil {
		logrus.Debugf("image %s already exists", imageName)
		return img, nil
	}
	return nil, nil
}

// Pull always do pull action
func (d DefaultImageService) Pull(imageName string, platforms []*v1.Platform) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}
	var (
		reader, writer  = io.Pipe()
		writeFlusher    = dockerioutils.NewWriteFlusher(writer)
		progressChanOut = streamformatter.NewJSONProgressOutput(writeFlusher, false)
		streamOut       = dockerstreams.NewOut(common.StdOut)
		ctx             = context.Background()
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

	repo, err := distributionutil.NewV2Repository(named, "pull")
	if err != nil {
		return err
	}

	manifest, err := repo.Manifests(ctx, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get manifest service error: %v", err)
	}
	desc, err := repo.Tags(ctx).Get(ctx, named.Tag())
	if err != nil {
		return fmt.Errorf("get %s tag descriptor error: %v, try \"docker login\" if you are using a private registry", named.Repo(), err)
	}

	puller, err := distributionutil.NewPuller(repo, distributionutil.Config{
		LayerStore:     layerStore,
		ProgressOutput: progressChanOut,
	})
	if err != nil {
		return err
	}

	go func() {
		err := dockerjsonmessage.DisplayJSONMessagesToStream(reader, streamOut, nil)
		if err != nil && err != io.ErrClosedPipe {
			logrus.Warnf("error occurs in display progressing, err: %s", err)
		}
	}()

	dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Start to Pull Image %s", named.Raw()))
	maniList, err := manifest.Get(ctx, desc.Digest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return fmt.Errorf("get image manifest error: %v", err)
	}
	_, p, err := maniList.Payload()
	if err != nil {
		return fmt.Errorf("failed to get image manifest list payload: %v", err)
	}
	for _, plat := range platforms {
		m, err := d.handleManifest(ctx, manifest, p, *plat)
		if err != nil {
			return fmt.Errorf("get digest error: %v", err)
		}

		image, err := puller.Pull(ctx, named, m)
		if err != nil {
			return err
		}

		err = d.imageStore.Save(*image)
		if err != nil {
			return err
		}
	}
	dockerprogress.Message(progressChanOut, "", fmt.Sprintf("Success to Pull Image %s", named.Raw()))
	return nil
}

func (d DefaultImageService) handleManifest(ctx context.Context, manifest distribution.ManifestService, payload []byte, platform v1.Platform) (schema2.Manifest, error) {
	dgest, err := distributionutil.GetImageManifestDigest(payload, platform)
	if err != nil {
		return schema2.Manifest{}, fmt.Errorf("get digest from manifest list error: %v", err)
	}

	m, err := manifest.Get(ctx, dgest, make([]distribution.ManifestServiceOption, 0)...)
	if err != nil {
		return schema2.Manifest{}, fmt.Errorf("get image manifest error: %v", err)
	}

	_, ok := m.(*schema2.DeserializedManifest)
	if !ok {
		return schema2.Manifest{}, fmt.Errorf("failed to parse manifest to DeserializedManifest")
	}
	return m.(*schema2.DeserializedManifest).Manifest, nil
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
			logrus.Warnf("error occurs in display progressing, err: %s", err)
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

	svc, err := auth.NewDockerAuthService()
	if err != nil {
		return fmt.Errorf("failed to read default auth file: %v", err)
	}

	if err := svc.SetAuthInfo(RegistryURL, RegistryUsername, RegistryPasswd); err != nil {
		return err
	}
	logrus.Infof("%s login %s success", RegistryUsername, RegistryURL)
	return nil
}

// Delete image layer, image meta, and local registry by id or name.
// if only image id,will delete related image data.
// if image name,and not specify platform,will delete all image data.
// if image name and  platforms is not nil will delete all the related platform images.
func (d DefaultImageService) Delete(imageNameOrID string, platforms []*v1.Platform) error {
	var (
		err               error
		imageStore        = d.imageStore
		isImageID         bool
		deleteImageIDList []string
	)

	imageMetadataMap, err := imageStore.GetImageMetadataMap()
	if err != nil {
		return err
	}

	// detect if the input is image id.
	for _, manifestList := range imageMetadataMap {
		for _, m := range manifestList.Manifests {
			if m.ID == imageNameOrID {
				isImageID = true
				break
			}
		}
	}

	if isImageID {
		// delete image by id
		err = imageStore.DeleteByID(imageNameOrID)
		if err != nil {
			return err
		}
		deleteImageIDList = append(deleteImageIDList, imageNameOrID)
	} else {
		// delete image by name
		if len(platforms) == 0 {
			// delete all platforms
			manifestList, err := imageStore.GetImageManifestList(imageNameOrID)
			if err != nil {
				return err
			}
			for _, m := range manifestList {
				deleteImageIDList = append(deleteImageIDList, m.ID)
			}
			if err = imageStore.DeleteByName(imageNameOrID, nil); err != nil {
				return fmt.Errorf("failed to delete image %s, err: %v", imageNameOrID, err)
			}
		} else {
			// delete user specify platform
			for _, plat := range platforms {
				img, err := imageStore.GetByName(imageNameOrID, plat)
				if err != nil {
					return fmt.Errorf("image %s not found %v", imageNameOrID, err)
				}

				if err = imageStore.DeleteByName(imageNameOrID, plat); err != nil {
					return fmt.Errorf("failed to delete image %s, err: %v", imageNameOrID, err)
				}
				deleteImageIDList = append(deleteImageIDList, img.Spec.ID)
			}
		}
	}

	// delete image.yaml file which id not in current imageMetadataMap.
	for _, id := range strings.RemoveDuplicate(deleteImageIDList) {
		err = store.DeleteImageLocal(id)
		if err != nil {
			return err
		}
	}

	err = d.deleteLayers()
	if err != nil {
		return err
	}
	logrus.Infof("image %s delete success", imageNameOrID)
	return nil
}

//delete the unused Layer in the `DefaultLayerDir` directory
func (d DefaultImageService) deleteLayers() error {
	var (
		//save a path with desired name as value.
		pruneMap = make(map[string][]string)
	)

	allImageLayerIDList, err := d.getAllLayers()
	if err != nil {
		return err
	}

	pruneMap[common.DefaultLayerDir] = allImageLayerIDList
	pruneMap[filepath.Join(common.DefaultLayerDBRoot, "sha256")] = allImageLayerIDList

	for root, desired := range pruneMap {
		subset, err := osi.GetDirNameListInDir(root, osi.FilterOptions{
			All:          true,
			WithFullPath: false,
		})
		if err != nil {
			return err
		}

		trash := strings.NewComparator(subset, desired).GetSrcSubtraction()
		for _, name := range trash {
			if err := os.RemoveAll(filepath.Join(root, name)); err != nil {
				return err
			}
			_, err := common.StdOut.WriteString(fmt.Sprintf("%s deleted\n", name))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// getAllLayers return current image id and layers
func (d DefaultImageService) getAllLayers() ([]string, error) {
	imageMetadataMap, err := d.imageStore.GetImageMetadataMap()
	var allImageLayerDirs []string

	if err != nil {
		return nil, err
	}

	for _, imageMetadata := range imageMetadataMap {
		for _, m := range imageMetadata.Manifests {
			image, err := d.imageStore.GetByID(m.ID)
			if err != nil {
				return nil, err
			}
			for _, layer := range image.Spec.Layers {
				if layer.ID != "" {
					allImageLayerDirs = append(allImageLayerDirs, layer.ID.Hex())
				}
			}
		}
	}
	return strings.RemoveDuplicate(allImageLayerDirs), err
}
