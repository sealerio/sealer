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
	"fmt"

	"github.com/opencontainers/go-digest"

	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/image/cache"
)

type Prober interface {
	Reset()
	Probe(parentID string, layer *cache.Layer) (cacheID digest.Digest, err error)
}

type imageProber struct {
	cache       Cache
	reset       func() Cache
	cacheBusted bool
}

func NewImageProber(cacheBuilder CacheBuilder, noCache bool) Prober {
	if noCache {
		return &nopProber{}
	}

	reset := func() Cache {
		c, err := cacheBuilder.BuildImageCache()
		if err != nil {
			logger.Info("failed to init image cache, err: %s", err)
			return &cache.NopImageCache{}
		}
		return c
	}

	return &imageProber{cache: reset(), reset: reset}
}

func (c *imageProber) Reset() {
	c.cache = c.reset()
	c.cacheBusted = false
}

func (c *imageProber) Probe(parentID string, layer *cache.Layer) (cacheID digest.Digest, err error) {
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

func (c *nopProber) Probe(_ string, _ *cache.Layer) (digest.Digest, error) {
	return "", fmt.Errorf("nop prober")
}
