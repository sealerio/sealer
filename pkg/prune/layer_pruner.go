package prune

import (
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/utils"
)

type layerPrune struct {
	layerDataDir string
	layerDBDir   string
	imageStore   store.ImageStore
}

func NewLayerPrune() (Selector, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return layerPrune{imageStore: imageStore,
		layerDataDir: common.DefaultLayerDir,
		layerDBDir:   filepath.Join(common.DefaultLayerDBRoot, "sha256"),
	}, nil
}

func (l layerPrune) Pickup() ([]string, error) {
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
			if utils.NotIn(filepath.Base(subset), desired) {
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
	return utils.RemoveDuplicate(allImageLayerDirs), err
}
func (l layerPrune) GetSelectorMessage() string {
	return LayerPruner
}
