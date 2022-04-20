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

package store

import (
	"fmt"

	"github.com/alibaba/sealer/pkg/image/reference"

	"github.com/alibaba/sealer/pkg/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type imageStore struct {
	backend Backend
}

func (is *imageStore) GetByName(name string, platform *v1.Platform) (*v1.Image, error) {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return nil, err
	}
	return is.backend.getImageByName(named.CompleteName(), platform)
}

func (is *imageStore) GetByID(id string) (*v1.Image, error) {
	return is.backend.getImageByID(id)
}

func (is *imageStore) DeleteByName(name string, platform *v1.Platform) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	return is.backend.deleteImage(named.CompleteName(), platform)
}

func (is *imageStore) DeleteByID(id string) error {
	return is.backend.deleteImageByID(id)
}

func (is *imageStore) Save(image v1.Image) error {
	return is.backend.saveImage(image)
}

func (is *imageStore) SetImageMetadataItem(name string, imageMetadata *types.ManifestDescriptor) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	return is.backend.setImageMetadata(named.CompleteName(), imageMetadata)
}

func (is *imageStore) GetImageMetadataItem(name string, platform *v1.Platform) (*types.ManifestDescriptor, error) {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return nil, err
	}
	return is.backend.getImageMetadataItem(named.CompleteName(), platform)
}

func (is *imageStore) GetImageMetadataMap() (ImageMetadataMap, error) {
	return is.backend.getImageMetadataMap()
}

func (is *imageStore) GetImageManifestList(name string) ([]*types.ManifestDescriptor, error) {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return nil, err
	}

	metadata, err := is.backend.getImageMetadataMap()
	if err != nil {
		return nil, err
	}

	if ml, ok := metadata[named.CompleteName()]; ok {
		return ml.Manifests, nil
	}

	return nil, fmt.Errorf("%s not found", name)
}

func NewDefaultImageStore() (ImageStore, error) {
	backend, err := NewFSStoreBackend()
	if err != nil {
		return nil, fmt.Errorf("failed to init fs store backend, err: %v", err)
	}

	return &imageStore{
		backend: backend,
	}, nil
}
