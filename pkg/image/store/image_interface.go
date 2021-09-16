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
	"github.com/alibaba/sealer/pkg/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ImageStore interface {
	GetByName(name string) (*v1.Image, error)

	GetByID(id string) (*v1.Image, error)

	DeleteByName(name string) error

	DeleteByID(id string, force bool) error

	Save(image v1.Image, name string) error

	SetImageMetadataItem(name, id string) error

	GetImageMetadataItem(name string) (types.ImageMetadata, error)

	GetImageMetadataMap() (ImageMetadataMap, error)
}
