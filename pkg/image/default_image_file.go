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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/pkg/image/reference"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/pkg/image/types"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/archive"
)

type DefaultImageFileService struct {
	layerStore store.LayerStore
	imageStore store.ImageStore
}

func (d DefaultImageFileService) Load(imageSrc string) error {
	imageMetadata, err := d.load(imageSrc)
	if err == nil {
		logger.Info("load image %s [id: %s] successfully", imageMetadata.Name, imageMetadata.ID)
	}
	return err
}

func (d DefaultImageFileService) Save(image *v1.Image, imageTar string) error {
	// imageTar maybe below value:
	// full name : /tmp/image.tar
	// only file name : image.tar
	// only file path: /tmp , /tmp/
	if imageTar == "" {
		return fmt.Errorf("imagetar cannot be empty")
	}

	dir, file := filepath.Split(imageTar)
	if dir == "" {
		dir = "."
	}
	if file == "" {
		file = fmt.Sprintf("%s.tar", image.Name)
	}
	imageTar = filepath.Join(dir, file)
	// only file path like "/tmp" will lose add image tar name,make sure imageTar with full file name.
	if filepath.Ext(imageTar) != ".tar" {
		imageTar = filepath.Join(imageTar, fmt.Sprintf("%s.tar", image.Name))
	}

	if err := utils.MkFileFullPathDir(imageTar); err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}

	return d.save(image, imageTar)
}

func (d DefaultImageFileService) Merge(image *v1.Image) error {
	panic("implement me")
}

func (d DefaultImageFileService) save(image *v1.Image, imageTar string) error {
	file, err := os.Create(filepath.Clean(imageTar))
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()
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
	imgBytes, err := yaml.Marshal(image)
	if err != nil {
		return fmt.Errorf("failed to marchal image, err: %s", err)
	}
	if err = utils.AtomicWriteFile(imageMetadataTempFile, imgBytes, common.FileMode0644); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", imageMetadataTempFile, err)
	}
	metadata, err := d.imageStore.GetImageMetadataItem(image.Name)
	if err != nil {
		return err
	}
	repo, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	if err = utils.AtomicWriteFile(repofile, repo, common.FileMode0644); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", imageMetadataTempFile, err)
	}

	pathsToCompress = append(pathsToCompress, imageMetadataTempFile, repofile)
	tarReader, err := archive.TarWithRootDir(pathsToCompress...)
	if err != nil {
		return fmt.Errorf("failed to get tar reader for %s, err: %s", image.Name, err)
	}
	defer tarReader.Close()

	_, err = io.Copy(file, tarReader)
	return err
}

func (d DefaultImageFileService) load(imageSrc string) (*types.ImageMetadata, error) {
	var (
		srcFile *os.File
		size    int64
		err     error
		image   v1.Image
	)
	srcFile, err = os.Open(filepath.Clean(imageSrc))
	if err != nil {
		return nil, fmt.Errorf("failed to open %s, err : %v", imageSrc, err)
	}
	defer func() {
		if err := srcFile.Close(); err != nil {
			logger.Fatal("failed to close file")
		}
	}()
	srcFi, err := srcFile.Stat()
	if err != nil {
		return nil, err
	}
	size = srcFi.Size()

	if _, err = archive.Decompress(srcFile, common.DefaultLayerDir, archive.Options{Compress: false}); err != nil {
		return nil, err
	}
	repofile := filepath.Join(common.DefaultLayerDir, common.DefaultMetadataName)
	defer os.Remove(repofile)

	repo, err := ioutil.ReadFile(filepath.Clean(repofile))
	if err != nil {
		return nil, err
	}
	var imageMetadata types.ImageMetadata

	if err := json.Unmarshal(repo, &imageMetadata); err != nil {
		return nil, err
	}

	imageTempFile := filepath.Join(common.DefaultLayerDir, common.DefaultImageMetadataFileName)
	defer os.Remove(imageTempFile)

	if err = utils.UnmarshalYamlFile(imageTempFile, &image); err != nil {
		return nil, fmt.Errorf("failed to parsing %s, err: %v", imageTempFile, err)
	}

	for _, layer := range image.Spec.Layers {
		if layer.ID == "" {
			continue
		}
		// TODO distributionMetadata
		roLayer, err := store.NewROLayer(layer.ID, size, nil)
		if err != nil {
			return nil, err
		}

		err = d.layerStore.RegisterLayerIfNotPresent(roLayer)
		if err != nil {
			return nil, fmt.Errorf("failed to register layer, err: %v", err)
		}
	}

	named, err := reference.ParseToNamed(imageMetadata.Name)
	if err != nil {
		return nil, err
	}

	return &imageMetadata, d.imageStore.Save(image, named.Raw())
}
