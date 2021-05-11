package image

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/reference"
	"github.com/alibaba/sealer/image/store"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/compress"
)

type DefaultImageFileService struct {
	BaseImageManager
}

func (d DefaultImageFileService) Load(imageSrc string) error {
	return d.load(imageSrc)
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
	ima, err := imageutils.GetImage(named.Raw())
	if err != nil {
		return err
	}
	file, err := os.Create(imageTar)
	if err != nil {
		return fmt.Errorf("failed to create %s, err: %v", imageTar, err)
	}
	defer file.Close()

	var pathsToCompress []string
	layerDirs, err := GetImageLayerDirs(ima)
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
	imageMetadataYaml := filepath.Join(common.DefaultImageMetaRootDir, ima.Spec.ID+common.YamlSuffix)
	imageMetadata, err := ioutil.ReadFile(imageMetadataYaml)
	if err != nil {
		return fmt.Errorf("failed to read %s, err: %v", imageMetadataYaml, err)
	}
	if err := ioutil.WriteFile(imageMetadataTempFile, imageMetadata, common.FileMode0644); err != nil {
		return fmt.Errorf("failed to write temp file %s, err: %v ", imageMetadataTempFile, err)
	}
	pathsToCompress = append(pathsToCompress, imageMetadataTempFile)
	_, err = compress.Compress(file, pathsToCompress...)
	return err
}

func (d DefaultImageFileService) load(imageSrc string) error {
	src, err := os.Open(imageSrc)
	if err != nil {
		return fmt.Errorf("failed to open %s, err : %v", imageSrc, err)
	}
	defer src.Close()
	if err := compress.Decompress(src, common.DefaultLayerDir); err != nil {
		return err
	}
	imageTempFile := filepath.Join(common.DefaultLayerDir, common.DefaultImageMetadataFileName)
	var image v1.Image
	if err := utils.UnmarshalYamlFile(imageTempFile, &image); err != nil {
		return fmt.Errorf("failed to parsing %s.yaml, err: %v", imageTempFile, err)
	}
	defer os.Remove(imageTempFile)
	layerStore, err := store.NewDefaultLayerStore()
	if err != nil {
		return fmt.Errorf("failed to get layerstore, err: %v", err)
	}
	for _, layer := range image.Spec.Layers {
		if layer.Hash != "" {
			roLayer, err := store.NewROLayer(layer.Hash, 0)
			if err != nil {
				return err
			}
			err = layerStore.RegisterLayerIfNotPresent(roLayer)
			if err != nil {
				return fmt.Errorf("failed to register layer, err: %v", err)
			}
		}
	}
	named, err := reference.ParseToNamed(image.Name)
	if err != nil {
		return err
	}
	return d.syncImageLocal(image, named)
}
