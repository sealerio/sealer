package image

import v1 "github.com/alibaba/sealer/types/api/v1"

type ImageCacheBuilder interface {
	BuildImageCache() ImageCache
}

type ImageCache interface {
	GetCache(parentID string, layer *v1.Layer) (LayerID string, err error)
}
