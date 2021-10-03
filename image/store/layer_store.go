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

package store

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/opencontainers/go-digest"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"

	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils/archive"
)

type layerStore struct {
	mux    sync.RWMutex
	layers map[LayerID]*ROLayer
	Backend
}

func (ls *layerStore) Get(id LayerID) Layer {
	ls.mux.RLock()
	defer ls.mux.RUnlock()
	l, ok := ls.layers[id]
	if !ok {
		return nil
	}
	return l
}

func (ls *layerStore) RegisterLayerIfNotPresent(layer Layer) error {
	layerExist := ls.Get(layer.ID())
	if layerExist != nil {
		return nil
	}

	curLayerDBDir := ls.LayerDBDir(layer.ID().ToDigest())
	err := os.MkdirAll(curLayerDBDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to init layer db for %s, err: %s", curLayerDBDir, err)
	}

	layerTarReader, err := layer.TarStream()
	if err != nil {
		return err
	}
	defer layerTarReader.Close()

	err = ls.DisassembleTar(layer.ID().ToDigest(), layerTarReader)
	if err != nil {
		return err
	}

	err = ls.storeROLayer(layer)
	if err != nil {
		return err
	}

	ls.mux.Lock()
	defer ls.mux.Unlock()
	if roLayer, ok := layer.(*ROLayer); ok {
		ls.layers[layer.ID()] = roLayer
	}

	return nil
}

func (ls *layerStore) RegisterLayerForBuilder(path string) (digest.Digest, error) {
	dist, size, err := archive.TarCanonicalDigest(path)
	if err != nil {
		return "", err
	}

	if dist == "" {
		return "", nil
	}

	// layerContentDigest is the layer id at the build stage.
	// and the layer id won't change any more
	roLayer, err := NewROLayer(dist, size, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create a new rolayer, err: %s", err)
	}
	layerDataDir := ls.LayerDataDir(roLayer.ID().ToDigest())

	// remove before mv files to target
	_, err = os.Stat(layerDataDir)
	if err == nil {
		err = os.RemoveAll(layerDataDir)
		if err != nil {
			return "", err
		}
	}

	err = os.Rename(path, layerDataDir)
	if err != nil {
		return "", err
	}

	return dist, ls.RegisterLayerIfNotPresent(roLayer)
}

func (ls *layerStore) DisassembleTar(layerID digest.Digest, streamReader io.ReadCloser) error {
	layerDBDir := ls.LayerDBDir(layerID)
	mf, err := os.OpenFile(filepath.Join(layerDBDir, tarDataGZ), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(0600))
	if err != nil {
		return err
	}
	defer mf.Close()

	mfz := gzip.NewWriter(mf)
	defer mfz.Close()

	metaPacker := storage.NewJSONPacker(mfz)
	// we're passing nil here for the file putter, because the ApplyDiff will
	// handle the extraction of the archive
	its, err := asm.NewInputTarStream(streamReader, metaPacker, nil)
	if err != nil {
		return err
	}

	_, err = io.Copy(ioutil.Discard, its)
	return err
}

func (ls *layerStore) Delete(id LayerID) error {
	digs := id.ToDigest()
	if layer := ls.Get(id); layer == nil {
		logger.Debug("layer %s is already deleted", id)
		return nil
	}

	layerDataPath := ls.LayerDataDir(digs)
	if err := os.RemoveAll(layerDataPath); err != nil {
		return err
	}

	layerDBDir := ls.LayerDBDir(digs)
	if err := os.RemoveAll(layerDBDir); err != nil {
		return err
	}

	ls.mux.Lock()
	defer ls.mux.Unlock()
	delete(ls.layers, id)
	return nil
}

func (ls *layerStore) AddDistributionMetadata(layerID LayerID, named reference.Named, descriptorDigest digest.Digest) error {
	return ls.addDistributionMetadata(layerID, map[string]digest.Digest{
		named.Domain() + "/" + named.Repo(): descriptorDigest,
	})
}

func (ls *layerStore) loadAllROLayers() error {
	roLayers, err := ls.Backend.loadAllROLayers()
	if err != nil {
		return fmt.Errorf("failed to load all layers, err: %s", err)
	}

	ls.mux.Lock()
	defer ls.mux.Unlock()
	for _, layer := range roLayers {
		ls.layers[layer.id] = layer
	}
	return nil
}

func NewDefaultLayerStore() (LayerStore, error) {
	sb, err := NewFSStoreBackend()
	if err != nil {
		return nil, err
	}

	ls := &layerStore{
		layers:  map[LayerID]*ROLayer{},
		Backend: sb,
	}
	err = ls.loadAllROLayers()
	if err != nil {
		return nil, err
	}
	return ls, nil
}
