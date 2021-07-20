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
	"github.com/alibaba/sealer/image/store"
	"github.com/alibaba/sealer/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

// MetadataService is the interface for providing image metadata service
type MetadataService interface {
	Tag(imageName, tarImageName string) error
	List() ([]types.ImageMetadata, error)
	GetImage(imageName string) (*v1.Image, error)
	GetRemoteImage(imageName string) (v1.Image, error)
	DeleteImage(imageName string) error
}

// FileService is the interface for file operations
type FileService interface {
	Load(imageSrc string) error
	Save(imageName string, imageTar string) error
	Merge(image *v1.Image) error
}

// Service is image service
type Service interface {
	Pull(imageName string) error
	PullIfNotExist(imageName string) error
	Push(imageName string) error
	Delete(imageName string) error
	Login(RegistryURL, RegistryUsername, RegistryPasswd string) error
	CacheBuilder
}

type LayerService interface {
	LayerStore() store.LayerStore
}
