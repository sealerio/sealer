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
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"
)

func upImageID(imageName string, Image *v1.Image) error {
	var (
		imageBytes []byte
		imageStore store.ImageStore
		err        error
	)
	imageBytes, err = yaml.Marshal(Image)
	if err != nil {
		return err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	Image.Spec.ID = imageID
	imageStore, err = store.NewDefaultImageStore()
	if err != nil {
		return err
	}
	return imageStore.Save(*Image, imageName)
}

func Merge(imageName string, images []string) error {
	var Image = &v1.Image{}
	for k, v := range images {
		img, err := DefaultImageService{}.PullIfNotExistAndReturnImage(v)
		if err != nil {
			return err
		}
		if k == 0 {
			Image = img
			Image.Name = imageName
			Image.Spec.Layers = img.Spec.Layers
		} else {
			Image.Spec.Layers = append(Image.Spec.Layers, img.Spec.Layers[1:]...)
		}
	}
	return upImageID(imageName, Image)
}
