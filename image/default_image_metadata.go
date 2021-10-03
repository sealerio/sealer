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

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"

	"github.com/alibaba/sealer/image/distributionutil"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	"github.com/alibaba/sealer/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

//DefaultImageMetadataService provide service for image metadata operations
type DefaultImageMetadataService struct {
	imageStore store.ImageStore
}

// Tag is used to give an another name for imageName
func (d DefaultImageMetadataService) Tag(imageName, tarImageName string) error {
	imageMetadata, err := d.imageStore.GetImageMetadataItem(imageName)
	if err != nil {
		return err
	}
	named, err := reference.ParseToNamed(tarImageName)
	if err != nil {
		return err
	}
	imageMetadata.Name = named.Raw()
	if err := d.imageStore.SetImageMetadataItem(imageMetadata); err != nil {
		return fmt.Errorf("failed to add tag %s, %s", tarImageName, err)
	}
	return nil
}

//List will list all kube-image locally
func (d DefaultImageMetadataService) List() ([]types.ImageMetadata, error) {
	imageMetadataMap, err := d.imageStore.GetImageMetadataMap()
	if err != nil {
		return nil, err
	}
	var imageMetadataList []types.ImageMetadata
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
	return d.imageStore.GetByName(imageName)
}

// GetRemoteImage will return the v1.Image from remote registry
func (d DefaultImageMetadataService) GetRemoteImage(imageName string) (v1.Image, error) {
	var (
		image v1.Image
		err   error
		named reference.Named
		ctx   = context.Background()
	)

	named, err = reference.ParseToNamed(imageName)
	if err != nil {
		return image, err
	}

	repo, err := distributionutil.NewV2Repository(named, "pull")
	if err != nil {
		return v1.Image{}, err
	}

	ms, err := repo.Manifests(ctx)
	if err != nil {
		return v1.Image{}, err
	}

	manifest, err := ms.Get(ctx, "", distribution.WithTagOption{Tag: named.Tag()})
	if err != nil {
		return v1.Image{}, err
	}

	// just transform it to schema2.DeserializedManifest
	// because we only upload this kind manifest.
	scheme2Manifest, ok := manifest.(*schema2.DeserializedManifest)
	if !ok {
		return v1.Image{}, fmt.Errorf("failed to parse manifest %s to DeserializedManifest", named.RepoTag())
	}

	bs := repo.Blobs(ctx)
	configJSONReader, err := bs.Open(ctx, scheme2Manifest.Config.Digest)
	if err != nil {
		return v1.Image{}, err
	}
	defer configJSONReader.Close()

	decoder := json.NewDecoder(configJSONReader)
	return image, decoder.Decode(&image)
}

func (d DefaultImageMetadataService) DeleteImage(imageName string) error {
	return d.imageStore.DeleteByName(imageName)
}
