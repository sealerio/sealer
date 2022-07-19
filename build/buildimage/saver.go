// Copyright © 2022 Alibaba Group Holding Ltd.
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

package buildimage

import (
	"fmt"

	imageUtils "github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/image/store"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

type imageSaver struct {
	platform   v1.Platform
	imageStore store.ImageStore
}

func (i imageSaver) Save(image *v1.Image) error {
	if err := i.setImageAttribute(image); err != nil {
		return err
	}

	if err := i.save(image); err != nil {
		return fmt.Errorf("failed to save image, err: %v", err)
	}
	return nil
}

func (i imageSaver) setImageAttribute(image *v1.Image) error {
	image.Spec.Platform = i.platform
	return NewAnnotationSetter().Set(image)
}

func (i imageSaver) save(image *v1.Image) error {
	imageID, err := imageUtils.GenerateImageID(*image)
	if err != nil {
		return err
	}
	image.Spec.ID = imageID
	return i.imageStore.Save(*image)
}

func NewImageSaver(platform v1.Platform) (ImageSaver, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}
	return imageSaver{
		imageStore: imageStore,
		platform:   platform,
	}, nil
}
