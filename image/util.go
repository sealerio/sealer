package image

import (
	"fmt"
	"github.com/docker/distribution"
	"github.com/opencontainers/go-digest"
	"gitlab.alibaba-inc.com/seadent/pkg/common"
	imageUtils "gitlab.alibaba-inc.com/seadent/pkg/image/utils"
	v1 "gitlab.alibaba-inc.com/seadent/pkg/types/api/v1"
	"gitlab.alibaba-inc.com/seadent/pkg/utils"
	"gitlab.alibaba-inc.com/seadent/pkg/utils/mount"
	"io/ioutil"
	"path/filepath"
)

func buildBlobs(dig digest.Digest, size int64, mediaType string) distribution.Descriptor {
	return distribution.Descriptor{
		Digest:    dig,
		Size:      size,
		MediaType: mediaType,
	}
}

func GetImageHashList(image *v1.Image) (res []string, err error) {
	baseLayer, err := GetImageAllLayers(image)
	if err != nil {
		return res, fmt.Errorf("get base image failed error is :%v\n", err)
	}
	for _, layer := range baseLayer {
		if layer.Hash != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.Hash))
		}
	}
	return
}

func GetImageAllLayers(image *v1.Image) (res []v1.Layer, err error) {
	for {
		res = append(res, image.Spec.Layers[1:]...)
		if image.Spec.Layers[0].Value == common.ImageScratch {
			break
		}
		if len(res) > 128 {
			return nil, fmt.Errorf("current layer is exceed 128 layers")
		}
		i, err := imageUtils.GetImage(image.Spec.Layers[0].Value)
		if err != nil {
			return []v1.Layer{}, err
		}
		image = i
	}
	return
}

func GetClusterFileFromImageName(imageName string) string {
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

func GetClusterFileFromBaseImage(imageName string) string {
	mountTarget, _ := utils.MkTmpdir()
	mountUpper, _ := utils.MkTmpdir()
	defer utils.CleanDirs(mountTarget, mountUpper)

	if err := NewImageService().PullIfNotExist(imageName); err != nil {
		return ""
	}
	driver := mount.NewMountDriver()
	image, err := imageUtils.GetImage(imageName)
	if err != nil {
		return ""
	}

	layers, err := GetImageHashList(image)
	if err != nil {
		return ""
	}

	err = driver.Mount(mountTarget, mountUpper, layers...)
	defer driver.Unmount(mountTarget)
	clusterFile := filepath.Join(mountTarget, "etc", common.DefaultClusterFileName)

	data, err := ioutil.ReadFile(clusterFile)
	if err != nil {
		return ""
	}
	return string(data)
}
