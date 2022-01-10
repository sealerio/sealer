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
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
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

func RemoveLayersDuplicate(list []v1.Layer) []v1.Layer {
	var result []v1.Layer
	flagMap := map[string]struct{}{}
	for _, v := range list {
		if _, ok := flagMap[v.ID.String()]; !ok {
			flagMap[v.ID.String()] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}

func Merge(imageName string, images []string) error {
	var (
		err        error
		Image      = &v1.Image{}
		img        *v1.Image
		imageStore store.ImageStore
		layers     []v1.Layer
	)
	imageStore, err = store.NewDefaultImageStore()
	if err != nil {
		return err
	}
	for k, v := range images {
		d := DefaultImageService{imageStore: imageStore}
		err = d.PullIfNotExist(v)
		if err != nil {
			return err
		}
		img, err = d.GetImageByName(v)
		if err != nil {
			return err
		}
		if k == 0 {
			Image = img
			Image.Name = imageName
			layers = img.Spec.Layers
		} else {
			layers = append(Image.Spec.Layers, img.Spec.Layers[1:]...)
		}
		Image.Spec.Layers = RemoveLayersDuplicate(layers)
	}
	return upImageID(imageName, Image)
}
