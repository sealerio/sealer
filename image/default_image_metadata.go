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
	"encoding/json"
	"fmt"
	"sort"

	"github.com/alibaba/sealer/registry"

	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

//DefaultImageMetadataService provide service for image metadata operations
type DefaultImageMetadataService struct {
}

// Tag is used to give an another name for imageName
func (d DefaultImageMetadataService) Tag(imageName, tarImageName string) error {
	imageMetadata, ok := imageutils.GetNewImageMetadata(imageName)
	if !ok {
		return fmt.Errorf("failed to found image %s", imageName)
	}
	named, err := reference.ParseToNamed(tarImageName)
	if err != nil {
		return err
	}
	imageMetadata.Name = named.Raw()
	if err := imageutils.SetImageMetadata(imageMetadata); err != nil {
		return fmt.Errorf("failed to add tag %s, %s", tarImageName, err)
	}
	return nil
}

//List will list all kube-image locally
func (d DefaultImageMetadataService) List() ([]imageutils.ImageMetadata, error) {
	imageMetadataMap, err := imageutils.GetImageMetadataMap()
	if err != nil {
		return nil, err
	}
	var imageMetadataList []imageutils.ImageMetadata
	for _, imageMetadata := range imageMetadataMap {
		imageMetadataList = append(imageMetadataList, imageMetadata)
	}
	sort.Slice(imageMetadataList, func(i, j int) bool {
		return imageMetadataList[i].Name < imageMetadataList[j].Name
	})
	return imageMetadataList, nil
}

// GetImage will return the v1.Image locally
func (d DefaultImageMetadataService) GetImage(imageName string) (*v1.Image, error) {
	return imageutils.GetImage(imageName)
}

// GetRemoteImage will return the v1.Image from remote registry
func (d DefaultImageMetadataService) GetRemoteImage(imageName string) (v1.Image, error) {
	var (
		image v1.Image
		err   error
		named reference.Named
		reg   *registry.Registry
	)

	named, err = reference.ParseToNamed(imageName)
	if err != nil {
		return image, err
	}

	reg, err = initRegistry(named.Domain())
	if err != nil {
		return image, err
	}

	manifest, err := reg.ManifestV2(context.Background(), named.Repo(), named.Tag())
	if err != nil {
		return image, err
	}

	configReader, err := reg.DownloadLayer(context.Background(), named.Repo(), manifest.Config.Digest)
	if err != nil {
		return image, err
	}

	decoder := json.NewDecoder(configReader)
	return image, decoder.Decode(&image)
}

func (d DefaultImageMetadataService) DeleteImage(imageName string) error {
	return imageutils.DeleteImage(imageName)
}
