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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/compress"
)

type DefaultImageFileService struct {
}

func (d DefaultImageFileService) Load(imageSrc string) error {
	imageMetadata, err := d.load(imageSrc)
	if err == nil {
		logger.Info("load image %s [id: %s] successfully", imageMetadata.Name, imageMetadata.ID)
	}
	return err
}

func (d DefaultImageFileService) Save(imageName string, imageTar string) error {
	if imageTar == "" {
		return fmt.Errorf("imagetar cannot be empty")
	}

	if utils.IsFileExist(imageTar) {
		return fmt.Errorf("file %s already exists", imageTar)
	}

	if err := utils.MkFileFullPathDir(imageTar); err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}

	return d.save(imageName, imageTar)
}

func (d DefaultImageFileService) Merge(image *v1.Image) error {
	panic("implement me")
}

func (d DefaultImageFileService) save(imageName, imageTar string) error {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return err
	}

	image, err := imageutils.GetImage(named.Raw())
	if err != nil {
		return err
	}
	file, err := os.Create(imageTar)
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}
	defer file.Close()

	var pathsToCompress []string
	layerDirs, err := GetImageLayerDirs(image)
	if err != nil {
		return err
	}
	pathsToCompress = append(pathsToCompress, layerDirs...)
	tempDir, err := utils.MkTmpdir()
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", tempDir, err)
	}
	defer utils.CleanDir(tempDir)
	imageMetadataTempFile := filepath.Join(tempDir, common.DefaultImageMetadataFileName)
	repofile := filepath.Join(tempDir, common.DefaultMetadataName)
	imageYaml := filepath.Join(common.DefaultImageMetaRootDir, image.Spec.ID+common.YamlSuffix)

	ima, err := ioutil.ReadFile(imageYaml)
	if err != nil {
		return fmt.Errorf("failed to read %s, err: %v", imageYaml, err)
	}
	if err = ioutil.WriteFile(imageMetadataTempFile, ima, common.FileMode0644); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", imageMetadataTempFile, err)
	}
	repo, err := json.Marshal(&imageutils.ImageMetadata{ID: image.Spec.ID, Name: named.Raw()})
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(repofile, repo, common.FileMode0644); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", imageMetadataTempFile, err)
	}

	pathsToCompress = append(pathsToCompress, imageMetadataTempFile, repofile)
	_, err = compress.Compress(file, pathsToCompress...)
	return err
}

func (d DefaultImageFileService) load(imageSrc string) (*imageutils.ImageMetadata, error) {
	src, err := os.Open(imageSrc)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, err : %v", imageSrc, err)
	}
	defer src.Close()
	if err = compress.Decompress(src, common.DefaultLayerDir); err != nil {
		return nil, err
	}
	repofile := filepath.Join(common.DefaultLayerDir, common.DefaultMetadataName)
	defer os.Remove(repofile)
	imageTempFile := filepath.Join(common.DefaultLayerDir, common.DefaultImageMetadataFileName)
	defer os.Remove(imageTempFile)
	var image v1.Image

	if err = utils.UnmarshalYamlFile(imageTempFile, &image); err != nil {
		return nil, fmt.Errorf("failed to parsing %s.yaml, err: %v", imageTempFile, err)
	}

	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return nil, fmt.Errorf("failed to get layerstore, err: %v", err)
	}

	for _, layer := range image.Spec.Layers {
		if layer.Hash == "" {
			continue
		}
		roLayer, err := store.NewROLayer(layer.Hash, 0)
		if err != nil {
			return nil, err
		}
		err = layerStore.RegisterLayerIfNotPresent(roLayer)
		if err != nil {
			return nil, fmt.Errorf("failed to register layer, err: %v", err)
		}
	}

	repo, err := ioutil.ReadFile(repofile)
	if err != nil {
		return nil, err
	}
	var imageMetadata imageutils.ImageMetadata

	if err := json.Unmarshal(repo, &imageMetadata); err != nil {
		return nil, err
	}
	named, err := reference.ParseToNamed(imageMetadata.Name)
	if err != nil {
		return nil, err
	}

	return &imageMetadata, store.SyncImageLocal(image, named)
}
