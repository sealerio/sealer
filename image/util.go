package image

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	imageUtils "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
	"github.com/docker/distribution"
	"github.com/opencontainers/go-digest"
)

func buildBlobs(dig digest.Digest, size int64, mediaType string) distribution.Descriptor {
	return distribution.Descriptor{
		Digest:    dig,
		Size:      size,
		MediaType: mediaType,
	}
}

// GetImageLayerDirs return image hash list
// current image is different with the image in build stage
// current image has no from layer
func GetImageLayerDirs(image *v1.Image) (res []string, err error) {
	for _, layer := range image.Spec.Layers {
		if layer.Type == common.FROMCOMMAND {
			return res, fmt.Errorf("image %s has from layer, which is not allowed in current state", image.Spec.ID)
		}
		if layer.Hash != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.Hash))
		}
	}
	return
}

// GetClusterFileFromImage retrieve ClusterFile From image
func GetClusterFileFromImage(imageName string) string {
	clusterfile := GetClusterFileFromImageManifest(imageName)
	if clusterfile != "" {
		return clusterfile
	}

	clusterfile = GetClusterFileFromBaseImage(imageName)
	if clusterfile != "" {
		return clusterfile
	}
	return ""
}

// GetClusterFileFromImageManifest retrieve ClusterFile from image manifest(image yaml)
func GetClusterFileFromImageManifest(imageName string) string {
	//  find cluster file from image manifest
	var image *v1.Image
	var err error
	image, err = imageUtils.GetImage(imageName)
	if err != nil {
		imageMetadata, err := NewImageMetadataService().GetRemoteImage(imageName)
		if err != nil {
			logger.Error("failed to find image %s,err: %v", imageName, err)
			return ""
		}
		image = &imageMetadata
	}
	Clusterfile, ok := image.Annotations[common.ImageAnnotationForClusterfile]
	if !ok {
		logger.Error("failed to find Clusterfile in local")
		return ""
	}
	return Clusterfile
}

// GetClusterFileFromBaseImage retrieve ClusterFile from base image, TODO need to refactor
func GetClusterFileFromBaseImage(imageName string) string {
	mountTarget, _ := utils.MkTmpdir()
	mountUpper, _ := utils.MkTmpdir()
	defer func() {
		utils.CleanDirs(mountTarget, mountUpper)
	}()

	if err := NewImageService().PullIfNotExist(imageName); err != nil {
		return ""
	}
	driver := mount.NewMountDriver()
	image, err := imageUtils.GetImage(imageName)
	if err != nil {
		return ""
	}

	layers, err := GetImageLayerDirs(image)
	if err != nil {
		return ""
	}

	err = driver.Mount(mountTarget, mountUpper, layers...)
	if err != nil {
		return ""
	}
	defer func() {
		err := driver.Unmount(mountTarget)
		if err != nil {
			logger.Warn(err)
		}
	}()

	clusterFile := filepath.Join(mountTarget, "etc", common.DefaultClusterFileName)
	data, err := ioutil.ReadFile(clusterFile)
	if err != nil {
		return ""
	}
	return string(data)
}

func GetYamlByImage(imageName string) (string, error) {
	imagesMap, err := imageUtils.GetImageMetadataMap()
	if err != nil {
		return "", err
	}

	image, ok := imagesMap[imageName]
	if !ok {
		return "", fmt.Errorf("failed to find image by name (%s)", imageName)
	}
	if image.ID == "" {
		return "", fmt.Errorf("failed to find corresponding image id, id is empty")
	}

	ImageInformation, err := ioutil.ReadFile(filepath.Join(common.DefaultImageMetaRootDir, image.ID+common.YamlSuffix))
	if err != nil {
		return "", fmt.Errorf("failed to read image yaml,err: %v", err)
	}
	return string(ImageInformation), nil
}
