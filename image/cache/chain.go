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

package cache

import (
	"sync"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"

	"github.com/alibaba/sealer/image/store"

	"github.com/alibaba/sealer/logger"

	"github.com/pkg/errors"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/opencontainers/go-digest"
)

var imageChain *chainStore
var once sync.Once

//ChainID is caculated from a series of serialized cache layers. The layers cacheID
// is "", but the COPY layer.
// same ChainID indicates that same entire file system.
type ChainID digest.Digest

func (id ChainID) String() string {
	return id.Digest().String()
}

// Digest converts ID into a digest
func (id ChainID) Digest() digest.Digest {
	return digest.Digest(id)
}

type ImageID digest.Digest

func (id ImageID) String() string {
	return id.Digest().String()
}

// Digest converts ID into a digest
func (id ImageID) Digest() digest.Digest {
	return digest.Digest(id)
}

// ChainStore is an interface for manipulating images
type ChainStore interface {
	Images() map[ImageID]*v1.Image
	GetChainLayer(id ChainID) (v1.Layer, error)
}

type chainItem struct {
	layer   v1.Layer
	chainID ChainID
}

type chainStore struct {
	sync.RWMutex
	chains map[ChainID]*chainItem
	fs     store.Backend
	ls     store.LayerStore
}

func NewImageStore(fs store.Backend, ls store.LayerStore) (ChainStore, error) {
	once.Do(func() {
		imageChain = &chainStore{
			chains: make(map[ChainID]*chainItem),
			fs:     fs,
			ls:     ls,
		}

		if err := imageChain.restore(); err != nil {
			return
		}
	})
	return imageChain, nil
}

// restore reads all images saved in filesystem and calculate their chainID
func (cs *chainStore) restore() error {
	cs.Lock()
	defer cs.Unlock()

	//read all image layers
	images := cs.Images()
	for _, image := range images {
		layers := image.Spec.Layers
		var lastChainItem = &chainItem{}
		for _, layer := range layers {
			var (
				chainID ChainID
				err     error
			)

			cacheLayer, err := cs.newCacheLayer(&layer)
			if err != nil {
				logger.Warn("failed to new a cache layer for %v, err: %s", layer, err)
				continue
			}

			// first chainItem's parent chainID is empty
			chainID, err = cacheLayer.ChainID(lastChainItem.chainID)
			if err != nil {
				logger.Error(err)
				break
			}
			logger.Debug("current layer %+v, restore chain id: %s", cacheLayer, chainID)

			_, ok := cs.chains[chainID]
			if !ok {
				cItem := &chainItem{
					layer:   layer,
					chainID: chainID,
				}
				cs.chains[chainID] = cItem
			}
			lastChainItem = &chainItem{
				layer:   layer,
				chainID: chainID,
			}
		}
	}

	return nil
}

func (cs *chainStore) GetChainLayer(id ChainID) (v1.Layer, error) {
	cs.RLock()
	defer cs.RUnlock()

	if imagemeta, ok := cs.chains[id]; ok {
		return imagemeta.layer, nil
	}

	return v1.Layer{}, errors.Errorf("no layer for chain id %s in file system", id)
}

func (cs *chainStore) Images() map[ImageID]*v1.Image {
	var (
		images  map[ImageID]*v1.Image
		configs [][]byte
		err     error
	)

	images = make(map[ImageID]*v1.Image)
	configs, err = cs.fs.ListImages()
	if err != nil {
		logger.Error("failed to get images from file system, err: %v", err)
		return nil
	}
	for _, config := range configs {
		img := &v1.Image{}
		err = yaml.Unmarshal(config, img)
		if err != nil {
			logger.Error("failed to unmarshal bytes into image")
			continue
		}
		dgst := digest.FromBytes(config)
		images[ImageID(dgst)] = img
	}

	return images
}

func (cs *chainStore) newCacheLayer(layer *v1.Layer) (*Layer, error) {
	var cacheLayer = Layer{Type: layer.Type, Value: layer.Value}
	// only copy layer needs the cache id.
	if layer.Type != common.COPYCOMMAND {
		return &cacheLayer, nil
	}

	cacheIDBytes, err := cs.fs.GetMetadata(layer.ID, common.CacheID)
	if err != nil {
		return nil, err
	}
	// TODO maybe we should validate the cacheid over digest
	cacheLayer.CacheID = string(cacheIDBytes)
	return &cacheLayer, nil
}

func CalculateCacheID(cacheLayers []Layer) (ChainID, error) {
	var parentID ChainID
	var err error

	for _, l := range cacheLayers {
		parentID, err = l.ChainID(parentID)
		if err != nil {
			return "", err
		}
	}

	return parentID, nil
}
