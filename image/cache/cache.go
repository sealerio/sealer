package cache

import (
	"strings"

	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/pkg/errors"
)

//LocalImageCache saves all the layer
type LocalImageCache struct {
	store store.Store
}

func NewLocalImageCache(store store.Store) *LocalImageCache {
	return &LocalImageCache{
		store: store,
	}
}

func (lic *LocalImageCache) GetCache(parentID string, layer *v1.Layer) (layerID string, err error) {
	tmpLayer, err := getLocalCachedImage(lic.store, store.StoreID(parentID), layer)
	if err != nil {
		return "", err
	}

	return tmpLayer.Hash.String(), err
}

func getLocalCachedImage(imageStore store.Store, parentID store.StoreID, layer *v1.Layer) (v1.Layer, error) {
	getMatch := func(siblings []store.StoreID) (v1.Layer, error) {
		var match v1.Layer
		for _, id := range siblings {
			targetLayer, err := imageStore.GetLayer(id)
			if err != nil {
				return v1.Layer{}, errors.Errorf("unable to find image %q", id)
			}

			if compare(&targetLayer, layer) {
				match = targetLayer
			}
		}

		if match.Hash == "" {
			return v1.Layer{}, errors.Errorf("unable to find image cache")
		}

		return match, nil
	}

	siblings := imageStore.Children(parentID)
	return getMatch(siblings)
}

func compare(a, b *v1.Layer) bool {
	if a == nil || b == nil {
		return false
	}

	if a.Type == b.Type &&
		strings.TrimSpace(a.Value) == strings.TrimSpace(b.Value) &&
		a.Hash != "" {
		return true
	}

	return false
}
