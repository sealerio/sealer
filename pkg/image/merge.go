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

	"github.com/alibaba/sealer/utils/platform"

	"github.com/alibaba/sealer/utils"

	"github.com/alibaba/sealer/common"
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
	image.Name = imageName
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
	return imageStore.Save(*image)
}

func Merge(imageName string, images []string) error {
	if imageName == "" {
		return fmt.Errorf("target image name should not be nil")
	}
	var (
		err        error
		newIma     *v1.Image
		imageStore store.ImageStore
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
			err = d.PullIfNotExist(im, platform.GetDefaultPlatform())
			if err != nil {
				return err
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}

	for i, v := range images {
		img, err := d.GetImageByName(v, platform.GetDefaultPlatform())
		if err != nil {
			return err
		}
		if i == 0 {
			newIma = img
			continue
		} else {
			newIma, err = merge(newIma, img)
			if err != nil {
				return err
			}
		}
	}
	return save(imageName, newIma)
}

func merge(base, ima *v1.Image) (*v1.Image, error) {
	if base == nil || ima == nil {
		return nil, fmt.Errorf(" merge base or new can not be nil")
	}
	// merge image platform
	if base.Spec.Platform.OS != ima.Spec.Platform.OS ||
		base.Spec.Platform.Architecture != ima.Spec.Platform.Architecture ||
		base.Spec.Platform.Variant != ima.Spec.Platform.Variant {
		return nil, fmt.Errorf("can not merge different platform")
	}

	var (
		isApp = base.Spec.ImageConfig.ImageType == common.AppImage &&
			ima.Spec.ImageConfig.ImageType == common.AppImage
	)
	// merge image type;only if two image is application image we can determine this new image is application image.
	if isApp {
		base.Spec.ImageConfig.ImageType = common.AppImage
	}

	// merge image config arg and remove duplicate value
	base.Spec.ImageConfig.Args = mergeImageArg(base.Spec.ImageConfig.Args, ima.Spec.ImageConfig.Args, isApp)
	// merge image config cmd and remove duplicate value
	base.Spec.ImageConfig.Cmd = mergeImageCmd(base.Spec.ImageConfig.Cmd, ima.Spec.ImageConfig.Cmd, isApp)

	// merge image layer
	res := append(base.Spec.Layers, ima.Spec.Layers...)
	base.Spec.Layers = removeDuplicateLayers(res)
	return base, nil
}

func mergeImageCmd(base, ima v1.ImageCmd, isApp bool) v1.ImageCmd {
	current := utils.MergeSlice(base.Current, ima.Current)
	if isApp {
		return v1.ImageCmd{
			Current: current,
		}
	}
	return v1.ImageCmd{
		Current: current,
		Parent:  utils.MergeSlice(base.Parent, ima.Parent),
	}
}

func mergeImageArg(base, ima v1.ImageArg, isApp bool) v1.ImageArg {
	for k, v := range ima.Current {
		base.Current[k] = v
	}

	if isApp {
		return v1.ImageArg{
			Current: base.Current,
		}
	}

	for k, v := range ima.Parent {
		base.Parent[k] = v
	}

	return v1.ImageArg{
		Parent:  base.Parent,
		Current: base.Current,
	}
}

func removeDuplicateLayers(list []v1.Layer) []v1.Layer {
	var result []v1.Layer
	flagMap := map[string]struct{}{}
	for _, v := range list {
		// if id is not nil,remove duplicate id,this covers run and copy instruction.
		if v.ID.String() != "" {
			if _, ok := flagMap[v.ID.String()]; !ok {
				flagMap[v.ID.String()] = struct{}{}
				result = append(result, v)
			}
		}
	}
	return result
}
