package image

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/docker/docker/api/types"

	"github.com/alibaba/sealer/common"
	imageUtils "github.com/alibaba/sealer/image/utils"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/mount"
)

// GetImageLayerDirs return image hash list
// current image is different with the image in build stage
// current image has no from layer
func GetImageLayerDirs(image *v1.Image) (res []string, err error) {
	for _, layer := range image.Spec.Layers {
		if layer.Type == common.FROMCOMMAND {
			return res, fmt.Errorf("image %s has from layer, which is not allowed in current state", image.Spec.ID)
		}
		if layer.Hash != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.Hash.Hex()))
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
	imageMetadata, err := NewImageMetadataService().GetRemoteImage(imageName)
	if err != nil {
		return ""
	}
	if imageMetadata.Annotations == nil {
		return ""
	}
	clusterFile, ok := imageMetadata.Annotations[common.ImageAnnotationForClusterfile]
	if !ok {
		return ""
	}

	return clusterFile
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

func getDockerAuthInfoFromDocker(domain string) types.AuthConfig {
	var (
		dockerInfo       *utils.DockerInfo
		err              error
		username, passwd string
	)
	dockerInfo, err = utils.DockerConfig()
	if err != nil {
		logger.Warn("failed to get docker info, err: %s", err)
	} else {
		username, passwd, err = dockerInfo.DecodeDockerAuth(domain)
		if err != nil {
			logger.Warn("failed to decode auth info, username and password would be empty, err: %s", err)
		}
	}

	return types.AuthConfig{Username: username, Password: passwd, ServerAddress: domain}
}
