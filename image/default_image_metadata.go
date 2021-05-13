package image

import (
	"context"
	"fmt"
	"sort"

	"github.com/alibaba/sealer/image/reference"
	imageutils "github.com/alibaba/sealer/image/utils"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

//DefaultImageMetadataService provide service for image metadata operations
type DefaultImageMetadataService struct {
	BaseImageManager
}

// Tag is used to give an another name for imageName
func (d DefaultImageMetadataService) Tag(imageName, tarImageName string) error {
	imageMetadataMap, err := imageutils.GetImageMetadataMap()
	if err != nil {
		return err
	}
	imageMetadata, ok := imageMetadataMap[imageName]
	if !ok {
		return fmt.Errorf("failed to found image %s", imageName)
	}
	imageMetadata.Name = tarImageName
	if err := imageutils.SetImageMetadata(imageMetadata); err != nil {
		return fmt.Errorf("failed to add tag %s, %s", tarImageName, err)
	}
	return nil
}

//List will list all kube-image locally
func (d DefaultImageMetadataService) List() ([]imageutils.ImageMetadata, error) {
	imageMetadataMap, err := imageutils.GetImageMetadataMap()
	if err != nil {
		return nil, err
	}
	var imageMetadataList []imageutils.ImageMetadata
	for _, imageMetadata := range imageMetadataMap {
		imageMetadataList = append(imageMetadataList, imageMetadata)
	}
	sort.Slice(imageMetadataList, func(i, j int) bool {
		return imageMetadataList[i].Name < imageMetadataList[j].Name
	})
	return imageMetadataList, nil
}

// GetImage will return the v1.Image locally
func (d DefaultImageMetadataService) GetImage(imageName string) (*v1.Image, error) {
	return imageutils.GetImage(imageName)
}

// GetRemoteImage will return the v1.Image from remote registry
func (d DefaultImageMetadataService) GetRemoteImage(imageName string) (v1.Image, error) {
	named, err := reference.ParseToNamed(imageName)
	if err != nil {
		return v1.Image{}, err
	}

	err = d.initRegistry(named.Domain())
	if err != nil {
		return v1.Image{}, err
	}

	manifest, err := d.registry.ManifestV2(context.Background(), named.Repo(), named.Tag())
	if err != nil {
		return v1.Image{}, err
	}

	return d.downloadImageManifestConfig(named, manifest.Config.Digest)
}

func (d DefaultImageMetadataService) DeleteImage(imageName string) error {
	return imageutils.DeleteImage(imageName)
}
