package image

import (
	"github.com/alibaba/sealer/image/cache"
	"github.com/alibaba/sealer/image/store"
	"github.com/alibaba/sealer/logger"
)

func (d DefaultImageService) BuildImageCache() ImageCache {
	ls, err := store.NewDefaultLayerStore()
	if err != nil {
		logger.Error("failed to build image cache")
		return nil
	}
	fs, err := store.NewFSStoreBackend("")
	if err != nil {
		logger.Error("failed to build image cache")
		return nil
	}
	imageStore, err := store.NewImageStore(fs, ls)
	if err != nil {
		logger.Error("failed to build image cache")
		return nil
	}

	return cache.NewLocalImageCache(imageStore)
}
