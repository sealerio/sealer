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
	"fmt"

	"github.com/alibaba/sealer/common"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/opencontainers/go-digest"
	"golang.org/x/sync/errgroup"
	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/pkg/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func save(imageName string, image *v1.Image) error {
	var (
		imageBytes []byte
		imageStore store.ImageStore
		err        error
	)
	imageStore, err = store.NewDefaultImageStore()
	if err != nil {
		return err
	}

	imageBytes, err = yaml.Marshal(image)
	if err != nil {
		return err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	image.Spec.ID = imageID
	err = setClusterFile(imageName, image)
	if err != nil {
		return err
	}
	return imageStore.Save(*image, imageName)
}

func setClusterFile(imageName string, image *v1.Image) error {
	var cluster v2.Cluster
	if image.Annotations == nil {
		return nil
	}
	raw := image.Annotations[common.ImageAnnotationForClusterfile]
	if err := yaml.Unmarshal([]byte(raw), &cluster); err != nil {
		return err
	}
	cluster.Spec.Image = imageName
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterData)
	return nil
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
	if imageName == "" {
		return fmt.Errorf("target image name should not be nil")
	}
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

	d := DefaultImageService{imageStore: imageStore}
	eg, _ := errgroup.WithContext(context.Background())

	for _, ima := range images {
		im := ima
		eg.Go(func() error {
			err = d.PullIfNotExist(im)
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for k, v := range images {
		img, err = d.GetImageByName(v)
		if err != nil {
			return err
		}
		if k == 0 {
			Image = img
			Image.Name = imageName
			layers = img.Spec.Layers
		} else {
			layers = append(Image.Spec.Layers, img.Spec.Layers...)
		}
		Image.Spec.Layers = RemoveLayersDuplicate(layers)
	}

	return save(imageName, Image)
}
