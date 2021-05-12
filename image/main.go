package image

import (
	"github.com/alibaba/sealer/image/store"
	"github.com/alibaba/sealer/logger"
)

var (
	globalLayerStore *store.LayerStore
)

func init() {
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		logger.Error("failed to init layer store, err: %s", err)
		panic(err)
	}
	globalLayerStore = &layerStore
}
