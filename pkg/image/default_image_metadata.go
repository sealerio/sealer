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

	"github.com/distribution/distribution/v3"
	"github.com/distribution/distribution/v3/manifest/schema2"

	"github.com/alibaba/sealer/pkg/image/distributionutil"
	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

//DefaultImageMetadataService provide service for image metadata operations
type DefaultImageMetadataService struct {
	imageStore store.ImageStore
}

// Tag is used to give a name for imageName
func (d DefaultImageMetadataService) Tag(imageName, tarImageName string) error {
	tarNamed, err := reference.ParseToNamed(tarImageName)
	if err != nil {
		return err
	}

	manifestList, err := d.imageStore.GetImageManifestList(imageName)
	if err != nil {
		return err
	}

	for _, m := range manifestList {
		image, err := d.imageStore.GetByID(m.ID)
		if err != nil {
			return err
		}

		image.Name = tarNamed.CompleteName()
		err = setClusterFile(tarNamed.CompleteName(), image)
		if err != nil {
			return err
		}

		imageID, err := GenerateImageID(*image)
		if err != nil {
			return err
		}
		image.Spec.ID = imageID
		err = d.imageStore.Save(*image)
		if err != nil {
			return err
		}
	}
	return nil
}

//List will list all cloud image locally
func (d DefaultImageMetadataService) List() (store.ImageMetadataMap, error) {
	return d.imageStore.GetImageMetadataMap()
}

// GetImage will return the v1.Image locally
func (d DefaultImageMetadataService) GetImage(imageName string, platform *v1.Platform) (*v1.Image, error) {
	image, err := d.imageStore.GetByName(imageName, platform)
	if err != nil {
		return nil, err
	}
	return image, nil
}

// GetRemoteImage will return the v1.Image from remote registry
func (d DefaultImageMetadataService) GetRemoteImage(imageName string, platform *v1.Platform) (v1.Image, error) {
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

func (d DefaultImageMetadataService) DeleteImage(imageName string, platform *v1.Platform) error {
	return d.imageStore.DeleteByName(imageName, platform)
}
