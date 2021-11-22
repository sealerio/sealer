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
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/pkg/ioutils"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/vbatts/tar-split/tar/asm"
	"github.com/vbatts/tar-split/tar/storage"
	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/types"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	pkgutils "github.com/alibaba/sealer/utils"
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
	getImageByName(name string) (*v1.Image, error)
	getImageByID(id string) (*v1.Image, error)
	deleteImage(name string) error
	deleteImageByID(id string, force bool) error
	saveImage(image v1.Image, name string) error
	setImageMetadata(metadata types.ImageMetadata) error
	getImageMetadataItem(nameOrID string) (types.ImageMetadata, error)
	getImageMetadataMap() (ImageMetadataMap, error)
}

type filesystem struct {
	sync.RWMutex
	layerDataRoot         string
	layerDBRoot           string
	imageDBRoot           string
	imageMetadataFilePath string
}

type ImageMetadataMap map[string]types.ImageMetadata

func NewFSStoreBackend() (Backend, error) {
	return &filesystem{
		layerDataRoot:         layerDataRoot,
		layerDBRoot:           layerDBRoot,
		imageDBRoot:           imageDBRoot,
		imageMetadataFilePath: imageMetadataFilePath,
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
		tarDataPath   = filepath.Join(fs.LayerDBDir(id.ToDigest()), tarDataGZ)
		layerDataPath = fs.LayerDataDir(id.ToDigest())
	)

	mf, err := os.Open(filepath.Clean(tarDataPath))
	if err != nil {
		return fmt.Errorf("failed to open %s for layer %s, err: %s", tarDataGZ, id, err)
	}

	mfz, err := gzip.NewReader(mf)
	if err != nil {
		err = mf.Close()
		if err != nil {
			return err
		}
		return err
	}

	gzipReader := ioutils.NewReadCloserWrapper(mfz, func() error {
		err := mfz.Close()
		if err != nil {
			return err
		}
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
	fileInfos, err = ioutil.ReadDir(fs.imageDBRoot)
	if err != nil {
		return nil, errors.Errorf("failed to open metadata directory %s, err: %v",
			fs.imageDBRoot, err)
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
	if err := os.MkdirAll(baseDir, common.FileMode0755); err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(baseDir, key), data, common.FileMode0644)
}

func (fs *filesystem) GetMetadata(id digest.Digest, key string) ([]byte, error) {
	fs.Lock()
	defer fs.Unlock()

	bytes, err := ioutil.ReadFile(filepath.Clean(filepath.Join(fs.LayerDBDir(id), key)))
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
	dig := layer.ID().ToDigest()
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

	idBytes, err := ioutil.ReadFile(filepath.Clean(filepath.Join(layerDBPath, "id")))
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

	sizeBytes, err := ioutil.ReadFile(filepath.Clean(filepath.Join(layerDBPath, "size")))
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

func (fs *filesystem) getImageMetadataMap() (ImageMetadataMap, error) {
	var (
		imagesMap ImageMetadataMap
	)
	// create file if not exists
	if !pkgutils.IsFileExist(fs.imageMetadataFilePath) {
		if err := pkgutils.WriteFile(fs.imageMetadataFilePath, []byte("{}")); err != nil {
			return nil, err
		}
		return ImageMetadataMap{}, nil
	}

	data, err := ioutil.ReadFile(fs.imageMetadataFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ImageMetadataMap, err: %s", err)
	}

	err = json.Unmarshal(data, &imagesMap)
	if err != nil {
		return nil, fmt.Errorf("failed to parsing ImageMetadataMap, err: %s", err)
	}
	return imagesMap, err
}

func (fs *filesystem) getImageByName(name string) (*v1.Image, error) {
	imagesMap, err := fs.getImageMetadataMap()
	if err != nil {
		return nil, err
	}
	//get an imageId based on the name of ClusterImage
	image, ok := imagesMap[name]
	if !ok {
		return nil, fmt.Errorf("failed to find image by name: %s", name)
	}

	if image.ID == "" {
		return nil, fmt.Errorf("failed to find corresponding image id, id is empty")
	}

	return fs.getImageByID(image.ID)
}

func (fs *filesystem) getImageByID(id string) (*v1.Image, error) {
	var (
		image    v1.Image
		filename = filepath.Join(fs.imageDBRoot, id+".yaml")
	)
	return &image, pkgutils.UnmarshalYamlFile(filename, &image)
}

func (fs *filesystem) deleteImage(name string) error {
	imagesMap, err := fs.getImageMetadataMap()
	if err != nil {
		return err
	}

	_, ok := imagesMap[name]
	if !ok {
		return nil
	}
	delete(imagesMap, name)

	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = pkgutils.AtomicWriteFile(fs.imageMetadataFilePath, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func (fs *filesystem) deleteImageByID(id string, force bool) error {
	imagesMap, err := fs.getImageMetadataMap()
	if err != nil {
		return err
	}
	var imageIDCount = 0
	var imageNames []string
	for _, value := range imagesMap {
		if value.ID == id {
			imageIDCount++
			imageNames = append(imageNames, value.Name)
		}
		if imageIDCount > 1 && !force {
			return fmt.Errorf("there are more than one image %s", id)
		}
	}
	if imageIDCount == 0 {
		return fmt.Errorf("failed to find image with id %s", id)
	}
	for _, imageName := range imageNames {
		delete(imagesMap, imageName)
	}

	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = pkgutils.AtomicWriteFile(fs.imageMetadataFilePath, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func (fs *filesystem) getImageMetadataItem(nameOrID string) (types.ImageMetadata, error) {
	imageMetadataMap, err := fs.getImageMetadataMap()
	imageMetadata := types.ImageMetadata{}
	if err != nil {
		return imageMetadata, err
	}
	for k, v := range imageMetadataMap {
		if nameOrID == k || nameOrID == v.ID {
			return v, nil
		}
	}
	return imageMetadata, &types.ImageNameOrIDNotFoundError{Name: nameOrID}
}

func (fs *filesystem) setImageMetadata(metadata types.ImageMetadata) error {
	metadata.CREATED = time.Now()
	imagesMap, err := fs.getImageMetadataMap()
	if err != nil {
		return err
	}

	imagesMap[metadata.Name] = metadata
	data, err := json.MarshalIndent(imagesMap, "", DefaultJSONIndent)
	if err != nil {
		return err
	}

	if err = pkgutils.AtomicWriteFile(fs.imageMetadataFilePath, data, common.FileMode0644); err != nil {
		return errors.Wrap(err, "failed to write DefaultImageMetadataFile")
	}
	return nil
}

func (fs *filesystem) saveImage(image v1.Image, name string) error {
	err := saveImageYaml(image, fs.imageDBRoot)
	if err != nil {
		return err
	}
	var res []string
	for _, layer := range image.Spec.Layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	size, err := pkgutils.GetFilesSize(res)
	if err != nil {
		return fmt.Errorf("failed to get image %s size, %v", name, err)
	}
	return fs.setImageMetadata(types.ImageMetadata{Name: name, ID: image.Spec.ID, SIZE: size})
}

func saveImageYaml(image v1.Image, dir string) error {
	imageYaml, err := yaml.Marshal(image)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dir, common.FileMode0755)
	if err != nil {
		return err
	}

	return pkgutils.AtomicWriteFile(filepath.Join(dir, image.Spec.ID+common.YamlSuffix), imageYaml, common.FileMode0644)
}
