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

	"github.com/alibaba/sealer/image/types"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type imageStore struct {
	backend Backend
}

func (is *imageStore) GetByName(name string) (*v1.Image, error) {
	return is.backend.getImageByName(name)
}

func (is *imageStore) GetByID(id string) (*v1.Image, error) {
	return is.backend.getImageByID(id)
}

func (is *imageStore) DeleteByName(name string) error {
	return is.backend.deleteImage(name)
}

func (is *imageStore) DeleteByID(id string, force bool) error {
	return is.backend.deleteImageByID(id, force)
}

func (is *imageStore) Save(image v1.Image, name string) error {
	return is.backend.saveImage(image, name)
}

func (is *imageStore) SetImageMetadataItem(name, id string) error {
	return is.backend.setImageMetadata(types.ImageMetadata{Name: name, ID: id})
}

func (is *imageStore) GetImageMetadataItem(name string) (types.ImageMetadata, error) {
	return is.backend.getImageMetadataItem(name)
}

func (is *imageStore) GetImageMetadataMap() (ImageMetadataMap, error) {
	return is.backend.getImageMetadataMap()
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
