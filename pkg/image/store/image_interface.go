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
	"github.com/sealerio/sealer/pkg/image/types"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

type ImageStore interface {
	GetByName(name string, platform *v1.Platform) (*v1.Image, error)

	GetByID(id string) (*v1.Image, error)

	DeleteByName(name string, platform *v1.Platform) error

	DeleteByID(id string) error

	Save(image v1.Image) error

	SetImageMetadataItem(name string, imageMetadata *types.ManifestDescriptor) error

	GetImageMetadataItem(name string, platform *v1.Platform) (*types.ManifestDescriptor, error)

	GetImageMetadataMap() (ImageMetadataMap, error)

	GetImageManifestList(name string) ([]*types.ManifestDescriptor, error)
}
