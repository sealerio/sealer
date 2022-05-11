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

type layerPrune struct {
	layerDataDir string
	layerDBDir   string
	imageStore   store.ImageStore
}

func NewLayerPrune() (Pruner, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return layerPrune{imageStore: imageStore,
		layerDataDir: common.DefaultLayerDir,
		layerDBDir:   filepath.Join(common.DefaultLayerDBRoot, "sha256"),
	}, nil
}

func (l layerPrune) Select() ([]string, error) {
	var (
		pruneList = make([]string, 0)
		//save a path with desired name as value.
		pruneMap = make(map[string][]string)
	)

	allLayerIDList, err := l.getAllLayers()
	if err != nil {
		return pruneList, err
	}

	pruneMap[l.layerDataDir] = allLayerIDList
	pruneMap[l.layerDBDir] = allLayerIDList

	for root, desired := range pruneMap {
		subsets, err := utils.GetDirNameListInDir(root, utils.FilterOptions{
			All:          true,
			WithFullPath: true,
		})
		if err != nil {
			return pruneList, err
		}
		for _, subset := range subsets {
			if slice.NotIn(filepath.Base(subset), desired) {
				pruneList = append(pruneList, subset)
			}
		}
	}

	return pruneList, nil
}

// getAllLayers return current image id and layers
func (l layerPrune) getAllLayers() ([]string, error) {
	imageMetadataMap, err := l.imageStore.GetImageMetadataMap()
	var allImageLayerDirs []string

	if err != nil {
		return nil, err
	}

	for _, imageMetadata := range imageMetadataMap {
		for _, m := range imageMetadata.Manifests {
			image, err := l.imageStore.GetByID(m.ID)
			if err != nil {
				return nil, err
			}
			for _, layer := range image.Spec.Layers {
				if layer.ID != "" {
					allImageLayerDirs = append(allImageLayerDirs, layer.ID.Hex())
				}
			}
		}
	}
	return slice.RemoveDuplicate(allImageLayerDirs), err
}
func (l layerPrune) GetSelectorMessage() string {
	return LayerPruner
}
