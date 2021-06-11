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
	"strings"
	"sync"

	"github.com/alibaba/sealer/logger"
	"github.com/docker/docker/pkg/ioutils"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"

	"github.com/alibaba/sealer/common"
	pkgutils "github.com/alibaba/sealer/utils"
	"github.com/pkg/errors"

	"github.com/opencontainers/go-digest"
)

const (
	imageDBRoot = common.DefaultImageDBRootDir
)

// Backend is a service for image/layer read and write.
// is majorly used by layer store.
// Avoid invoking backend by others as possible as we can.
type Backend interface {
	Get(id digest.Digest) ([]byte, error)
	Set(data []byte) (digest.Digest, error)
	Delete(id digest.Digest) error
	ListImages() ([][]byte, error)
	SetMetadata(id digest.Digest, key string, data []byte) error
	GetMetadata(id digest.Digest, key string) ([]byte, error)
	DeleteMetadata(id digest.Digest, key string) error
	LayerDBDir(digest digest.Digest) string
	LayerDataDir(digest digest.Digest) string
	assembleTar(id LayerID, writer io.Writer) error
	storeROLayer(layer Layer) error
	loadAllROLayers() ([]*ROLayer, error)
	addDistributionMetadata(layerID LayerID, newMetadatas map[string]digest.Digest) error
}

type filesystem struct {
	sync.RWMutex
	layerDataRoot string
	layerDBRoot   string
}

func NewFSStoreBackend() (Backend, error) {
	return &filesystem{
		layerDataRoot: layerDataRoot,
		layerDBRoot:   layerDBRoot,
	}, nil
}

func metadataDir(v interface{}) string {
	switch val := v.(type) {
	case digest.Digest:
		return filepath.Join(imageDBRoot, val.Hex()+common.YamlSuffix)
	case string:
		if strings.Contains(val, common.YamlSuffix) {
			return filepath.Join(imageDBRoot, val)
		}
		return filepath.Join(imageDBRoot, val+common.YamlSuffix)
	}

	return ""
}

func (fs *filesystem) Get(id digest.Digest) ([]byte, error) {
	var (
		metadata []byte
		err      error
	)
	fs.RLock()
	defer fs.RUnlock()

	//we do not use the functions in pkgutils because the validation steps
	//in its function is redundant in this situation
	metadata, err = ioutil.ReadFile(metadataDir(id))
	if err != nil {
		return nil, errors.Errorf("failed to read image %s's metadata, err: %v", id, err)
	}

	if digest.FromBytes(metadata) != id {
		return nil, errors.Errorf("failed to verify image %s's hash value", id)
	}

	return metadata, nil
}

func (fs *filesystem) Set(data []byte) (digest.Digest, error) {
	var (
		dgst digest.Digest
		err  error
	)
	fs.Lock()
	defer fs.Unlock()

	if len(data) == 0 {
		return "", errors.Errorf("invalid empty data")
	}

	dgst = digest.FromBytes(data)
	if err = ioutil.WriteFile(metadataDir(dgst), data, common.FileMode0644); err != nil {
		return "", errors.Errorf("failed to write image %s's metadata, err: %v", dgst, err)
	}

	return dgst, nil
}

func (fs *filesystem) Delete(dgst digest.Digest) error {
	var (
		err error
	)
	fs.Lock()
	defer fs.Unlock()

	if err = os.RemoveAll(metadataDir(dgst)); err != nil {
		return errors.Errorf("failed to delete image metadata, err: %v", err)
	}

	return nil
}

func (fs *filesystem) assembleTar(id LayerID, writer io.Writer) error {
	var (
		tarDataPath   = filepath.Join(fs.LayerDBDir(digest.Digest(id)), tarDataGZ)
		layerDataPath = fs.LayerDataDir(digest.Digest(id))
	)

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

func (fs *filesystem) ListImages() ([][]byte, error) {
	var (
		configs   [][]byte
		err       error
		fileInfos []os.FileInfo
	)
	fileInfos, err = ioutil.ReadDir(imageDBRoot)
	if err != nil {
		return nil, errors.Errorf("failed to open metadata directory %s, err: %v",
			imageDBRoot, err)
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			continue
		}

		if strings.Contains(fileInfo.Name(), common.YamlSuffix) {
			config, err := ioutil.ReadFile(metadataDir(fileInfo.Name()))
			if err != nil {
				logger.Error("failed to read file %v, err: %v", fileInfo.Name(), err)
			}
			configs = append(configs, config)
		}
	}

	return configs, nil
}

func (fs *filesystem) SetMetadata(id digest.Digest, key string, data []byte) error {
	fs.Lock()
	defer fs.Unlock()

	baseDir := fs.LayerDBDir(id)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(baseDir, key), data, 0644)
}

func (fs *filesystem) GetMetadata(id digest.Digest, key string) ([]byte, error) {
	fs.Lock()
	defer fs.Unlock()

	bytes, err := ioutil.ReadFile(filepath.Join(fs.LayerDBDir(id), key))
	if err != nil {
		return nil, errors.Errorf("failed to read metadata, err: %v", err)
	}

	return bytes, nil
}

func (fs *filesystem) DeleteMetadata(id digest.Digest, key string) error {
	fs.Lock()
	defer fs.Unlock()

	return os.RemoveAll(filepath.Join(fs.LayerDBDir(id), key))
}

func (fs *filesystem) LayerDBDir(digest digest.Digest) string {
	return filepath.Join(fs.layerDBRoot, digest.Algorithm().String(), digest.Hex())
}

func (fs *filesystem) LayerDataDir(digest digest.Digest) string {
	return filepath.Join(fs.layerDataRoot, digest.Hex())
}

func (fs *filesystem) storeROLayer(layer Layer) error {
	dig := digest.Digest(layer.ID())
	dbDir := fs.LayerDBDir(dig)
	err := pkgutils.WriteFile(filepath.Join(dbDir, "size"), []byte(fmt.Sprintf("%d", layer.Size())))
	if err != nil {
		return fmt.Errorf("failed to write size for %s, err: %s", layer.ID(), err)
	}

	err = fs.addDistributionMetadata(layer.ID(), layer.DistributionMetadata())
	if err != nil {
		return fmt.Errorf("failed to write distribution metadata for %s, err: %s", layer.ID(), err)
	}

	err = pkgutils.WriteFile(filepath.Join(dbDir, "id"), []byte(layer.ID()))
	logger.Debug("writing id %s to %s", layer.ID(), filepath.Join(dbDir, "id"))
	if err != nil {
		return fmt.Errorf("failed to write id for %s, err: %s", layer.ID(), err)
	}

	return nil
}

func (fs *filesystem) loadLayerID(layerDBPath string) (LayerID, error) {
	fs.RLock()
	defer fs.RUnlock()

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

func (fs *filesystem) loadLayerSize(layerDBPath string) (int64, error) {
	fs.RLock()
	defer fs.RUnlock()

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

func (fs *filesystem) loadROLayer(layerDBPath string) (*ROLayer, error) {
	layerID, err := fs.loadLayerID(layerDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get layer metadata %s, whose id file lost, err: %s", filepath.Base(layerDBPath), err)
	}

	layerSize, err := fs.loadLayerSize(layerDBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read size of layer %s, err: %s", filepath.Base(layerDBPath), err)
	}

	metadataMap, err := fs.LoadDistributionMetadata(layerID)
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

func (fs *filesystem) loadAllROLayers() ([]*ROLayer, error) {
	layerDirs, err := traverseLayerDB(fs.layerDBRoot)
	if err != nil {
		return nil, err
	}

	var layers []*ROLayer
	for _, layerDBDir := range layerDirs {
		rolayer, err := fs.loadROLayer(layerDBDir)
		if err != nil {
			logger.Warn(err)
			continue
		}
		layers = append(layers, rolayer)
	}
	return layers, nil
}
