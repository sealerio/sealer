package prune

import (
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/utils"
)

type imagePrune struct {
	imageRootDir string
	imageStore   store.ImageStore
}

func NewImagePrune() (Selector, error) {
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return imagePrune{imageStore: imageStore,
		imageRootDir: common.DefaultImageDBRootDir,
	}, nil
}

func (i imagePrune) Pickup() ([]string, error) {
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
		if utils.NotIn(filepath.Base(subset), imageIDFiles) {
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
