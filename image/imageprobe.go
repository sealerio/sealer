package image

import (
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ImageProber interface {
	Reset()
	Probe(parentID string, layer *v1.Layer) (cacheID string, err error)
}

type imageProber struct {
	cache       ImageCache
	reset       func() ImageCache
	cacheBusted bool
}

func NewImageProber(cacheBuilder ImageCacheBuilder, noCache bool) ImageProber {
	if noCache {
		return &nopProber{}
	}

	reset := func() ImageCache {
		return cacheBuilder.BuildImageCache()
	}

	return &imageProber{cache: reset(), reset: reset}
}

func (c *imageProber) Reset() {
	c.cache = c.reset()
	c.cacheBusted = false
}

func (c *imageProber) Probe(parentID string, layer *v1.Layer) (cacheID string, err error) {
	if c.cacheBusted {
		return "", nil
	}

	cacheID, err = c.cache.GetCache(parentID, layer)
	if err != nil {
		return "", err
	}

	return cacheID, nil
}

type nopProber struct{}

func (c *nopProber) Reset() {}

func (c *nopProber) Probe(_ string, _ *v1.Layer) (string, error) {
	return "", nil
}
