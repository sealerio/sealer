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

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type imageSaver struct {
	buildType  string
	imageStore store.ImageStore
}

func (i imageSaver) Save(image *v1.Image) error {
	err := i.setImageAttribute(image)
	if err != nil {
		return err
	}
	err = i.save(image)
	if err != nil {
		return fmt.Errorf("failed to save image, err: %v", err)
	}

	logger.Info("save image %s success", image.Name)
	return nil
}

func (i imageSaver) setImageAttribute(image *v1.Image) error {
	mi, err := GetLayerMountInfo(image.Spec.Layers, i.buildType)
	if err != nil {
		return err
	}
	defer mi.CleanUp()

	rootfsPath := mi.GetMountTarget()
	is := []ImageSetter{NewAnnotationSetter(rootfsPath), NewPlatformSetter(rootfsPath)}
	for _, s := range is {
		if err = s.Set(image); err != nil {
			return err
		}
	}
	return nil
}

func (i imageSaver) save(image *v1.Image) error {
	imageID, err := generateImageID(*image)
	if err != nil {
		return err
	}
	image.Spec.ID = imageID
	return i.imageStore.Save(*image, image.Name)
}

func NewImageSaver(buildType string) (ImageSaver, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}
	return imageSaver{
		buildType:  buildType,
		imageStore: imageStore,
	}, nil
}
