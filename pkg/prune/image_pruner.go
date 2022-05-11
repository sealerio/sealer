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

package prune

import (
	"path/filepath"

	"github.com/sealerio/sealer/utils/slice"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/store"
	"github.com/sealerio/sealer/utils"
)

type imagePrune struct {
	imageRootDir string
	imageStore   store.ImageStore
}

func NewImagePrune() (Pruner, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return imagePrune{imageStore: imageStore,
		imageRootDir: common.DefaultImageDBRootDir,
	}, nil
}

func (i imagePrune) Select() ([]string, error) {
	var pruneList []string

	imageIDFiles, err := i.getAllImageID()
	if err != nil {
		return pruneList, err
	}

	subsets, err := utils.GetDirNameListInDir(i.imageRootDir, utils.FilterOptions{
		All:          true,
		WithFullPath: true,
	})

	if err != nil {
		return pruneList, err
	}

	for _, subset := range subsets {
		if slice.NotIn(filepath.Base(subset), imageIDFiles) {
			pruneList = append(pruneList, subset)
		}
	}

	return pruneList, nil
}

// getAllImageID return current image id with ext "yaml".
func (i imagePrune) getAllImageID() ([]string, error) {
	imageMetadataMap, err := i.imageStore.GetImageMetadataMap()

	var imageIDList []string
	if err != nil {
		return nil, err
	}

	for _, imageMetadata := range imageMetadataMap {
		for _, m := range imageMetadata.Manifests {
			imageIDList = append(imageIDList, m.ID+".yaml")
		}
	}
	return imageIDList, err
}
func (i imagePrune) GetSelectorMessage() string {
	return ImagePruner
}
