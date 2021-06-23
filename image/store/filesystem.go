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
	"strconv"

	"github.com/docker/docker/pkg/ioutils"

	"github.com/alibaba/sealer/logger"

	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"

	"github.com/alibaba/sealer/utils"

	"github.com/opencontainers/go-digest"
)

type LayerStorage struct {
	LayerDataRoot string
	LayerDBRoot   string
}

func (ls LayerStorage) LayerDBDir(digest digest.Digest) string {
	return filepath.Join(ls.LayerDBRoot, digest.Algorithm().String(), digest.Hex())
}

func (ls LayerStorage) LayerDataDir(digest digest.Digest) string {
	return filepath.Join(ls.LayerDataRoot, digest.Hex())
}

func (ls LayerStorage) LoadLayerID(layerDBPath string) (LayerID, error) {
	idBytes, err := ioutil.ReadFile(filepath.Join(layerDBPath, "id"))
	if err != nil {
		return "", err
	}
	dig, err := digest.Parse(string(idBytes))
	if err != nil {
		return "", err
	}
	return LayerID(dig), nil
}

func (LayerStorage) LoadLayerSize(layerDBPath string) (int64, error) {
	sizeBytes, err := ioutil.ReadFile(filepath.Join(layerDBPath, "size"))
	if err != nil {
		return 0, err
	}

	size, err := strconv.ParseInt(string(sizeBytes), 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func (ls LayerStorage) loadROLayer(layerDBPath string) (*ROLayer, error) {
	layerID, err := ls.LoadLayerID(layerDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get layer metadata %s, whose id file lost, err: %s", filepath.Base(layerDBPath), err)
	}

	layerSize, err := ls.LoadLayerSize(layerDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read size of layer %s, err: %s", filepath.Base(layerDBPath), err)
	}

	metadataMap, err := ls.LoadDistributionMetadata(layerID)
	if err != nil {
		// we could tolerate the miss of DistributionMetadata.
		// the consequence is that we push the layer repeatedly
		logger.Warn("failed to get layer distribution digest, err: %s", filepath.Base(layerDBPath), err)
	}

	return NewROLayer(
		layerID.ToDigest(),
		layerSize,
		metadataMap,
	)
}

func (ls LayerStorage) storeROLayer(layer Layer) error {
	dig := digest.Digest(layer.ID())
	dbDir := ls.LayerDBDir(dig)
	err := utils.WriteFile(filepath.Join(dbDir, "size"), []byte(fmt.Sprintf("%d", layer.Size())))
	if err != nil {
		return fmt.Errorf("failed to write size for %s, err: %s", layer.ID(), err)
	}

	err = ls.addDistributionMetadata(layer.ID(), layer.DistributionMetadata())
	if err != nil {
		return fmt.Errorf("failed to write distribution metadata for %s, err: %s", layer.ID(), err)
	}

	err = utils.WriteFile(filepath.Join(dbDir, "id"), []byte(layer.ID()))
	logger.Debug("writing id %s to %s", layer.ID(), filepath.Join(dbDir, "id"))
	if err != nil {
		return fmt.Errorf("failed to write id for %s, err: %s", layer.ID(), err)
	}

	return nil
}

func (ls LayerStorage) assembleTar(id LayerID, writer io.Writer) error {
	var (
		tarDataPath   = filepath.Join(ls.LayerDBDir(digest.Digest(id)), tarDataGZ)
		layerDataPath = ls.LayerDataDir(digest.Digest(id))
	)
	_, err := os.Stat(tarDataPath)
	if err != nil {
		return fmt.Errorf("failed to find %s for layer %s, err: %s", tarDataGZ, id, err)
	}

	mf, err := os.Open(tarDataPath)
	if err != nil {
		return fmt.Errorf("failed to open %s for layer %s, err: %s", tarDataGZ, id, err)
	}

	mfz, err := gzip.NewReader(mf)
	if err != nil {
		mf.Close()
		return err
	}

	gzipReader := ioutils.NewReadCloserWrapper(mfz, func() error {
		mfz.Close()
		return mf.Close()
	})

	defer gzipReader.Close()
	metaUnpacker := storage.NewJSONUnpacker(gzipReader)
	fileGetter := storage.NewPathFileGetter(layerDataPath)
	return asm.WriteOutputTarStream(fileGetter, metaUnpacker, writer)
}

func NewLayerStorage(layerDataRoot, layerDBRoot string) LayerStorage {
	return LayerStorage{
		LayerDataRoot: layerDataRoot,
		LayerDBRoot:   layerDBRoot,
	}
}

func NewDefaultLayerStorage() LayerStorage {
	return NewLayerStorage(layerDataRoot, layerDBRoot)
}
