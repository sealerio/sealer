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

	"github.com/alibaba/sealer/utils/archive"

	"github.com/alibaba/sealer/image/reference"

	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"github.com/opencontainers/go-digest"
)

const emptySHA256TarDigest = "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

type layerStore struct {
	mux          sync.RWMutex
	layers       map[LayerID]*ROLayer
	layerStoreFS LayerStorage
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

	curLayerDBDir := ls.layerStoreFS.LayerDBDir(layer.ID().ToDigest())
	err := utils.MkDirIfNotExists(curLayerDBDir)
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

	err = ls.layerStoreFS.storeROLayer(layer)
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

func (ls *layerStore) RegisterLayerForBuilder(diffPath string) (digest.Digest, error) {
	tarReader, err := archive.TarWithoutRootDir(nil, diffPath)
	if err != nil {
		return "", fmt.Errorf("unable to tar on %s, err: %s", diffPath, err)
	}
	defer tarReader.Close()

	digester := digest.Canonical.Digester()
	size, err := io.Copy(digester.Hash(), tarReader)
	if err != nil {
		return "", err
	}
	layerDigest := digester.Digest()
	if layerDigest == emptySHA256TarDigest {
		return "", nil
	}

	// layerContentDigest is the layer id at the build stage.
	// and the layer id won't change any more
	roLayer, err := NewROLayer(layerDigest, size, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create a new rolayer, err: %s", err)
	}
	layerDataDir := ls.layerStoreFS.LayerDataDir(roLayer.ID().ToDigest())

	// TODO need a daemon to mk those dir at the sealer start
	err = utils.MkDirIfNotExists(layerDataRoot)
	if err != nil {
		return "", err
	}

	// remove before mv files to target
	_, err = os.Stat(layerDataDir)
	if err == nil {
		err = os.RemoveAll(layerDataDir)
		if err != nil {
			return "", err
		}
	}

	err = os.Rename(diffPath, layerDataDir)
	if err != nil {
		return "", err
	}

	return layerDigest, ls.RegisterLayerIfNotPresent(roLayer)
}

func (ls *layerStore) DisassembleTar(layerID digest.Digest, streamReader io.ReadCloser) error {
	layerDBDir := ls.layerStoreFS.LayerDBDir(layerID)
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
	layer := ls.Get(id)
	if layer == nil {
		logger.Debug("layer %s is already deleted", id)
		return nil
	}

	storefs := NewDefaultLayerStorage()
	layerDataPath := storefs.LayerDataDir(digs)
	err := os.RemoveAll(layerDataPath)
	if err != nil {
		return err
	}
	layerDBDir := storefs.LayerDBDir(digs)
	err = os.RemoveAll(layerDBDir)
	if err != nil {
		return err
	}
	ls.mux.Lock()
	defer ls.mux.Unlock()
	delete(ls.layers, id)
	return nil
}

func (ls *layerStore) AddDistributionMetadata(layerID LayerID, named reference.Named, descriptorDigest digest.Digest) error {
	sfs := ls.layerStoreFS
	return sfs.addDistributionMetadata(layerID, map[string]digest.Digest{
		named.Domain() + "/" + named.Repo(): descriptorDigest,
	})
}

func (ls *layerStore) loadAllROLayers() error {
	err := utils.MkDirIfNotExists(layerDBRoot)
	if err != nil {
		return err
	}

	layerDirs, err := traverseLayerDB()
	if err != nil {
		return err
	}

	var layers []*ROLayer
	sfs := ls.layerStoreFS
	for _, layerDBDir := range layerDirs {
		rolayer, err := sfs.loadROLayer(layerDBDir)
		if err != nil {
			logger.Warn(err)
			continue
		}
		layers = append(layers, rolayer)
	}

	ls.mux.Lock()
	defer ls.mux.Unlock()
	//TODO only check .../layerdb/.../id for existence of layer currently
	for _, layer := range layers {
		ls.layers[layer.id] = layer
	}
	return nil
}

func NewDefaultLayerStore() (LayerStore, error) {
	ls := &layerStore{
		layers:       map[LayerID]*ROLayer{},
		layerStoreFS: NewDefaultLayerStorage(),
	}
	err := ls.loadAllROLayers()
	if err != nil {
		return nil, err
	}
	return ls, nil
}
