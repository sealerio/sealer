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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/image/store"

	"github.com/alibaba/sealer/common"
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
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return
}

// GetClusterFileFromImage retrieves ClusterFile From image.
func GetClusterFileFromImage(imageName string) (string, error) {
	clusterFile, err := GetClusterFileFromImageManifest(imageName)
	if err != nil {
		return GetFileFromBaseImage(imageName, "etc", common.DefaultClusterFileName)
	}

	return clusterFile, nil
}

// GetClusterFileFromImageManifest retrieve ClusterFiles from image manifest(image yaml).
// When it runs into an error, returns a detailed error.
// When content getted is empty, returns an empty error directly to avoid upper caller to
// decide whether it is an empty.
func GetClusterFileFromImageManifest(imageName string) (string, error) {
	//  find cluster file from image manifest
	var (
		image *v1.Image
		err   error
	)
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return "", fmt.Errorf("failed to init image store: %v", err)
	}
	image, err = is.GetByName(imageName)
	if err != nil {
		ims, err := NewImageMetadataService()
		if err != nil {
			return "", fmt.Errorf("failed to create image metadata svcs: %v", err)
		}

		imageMetadata, err := ims.GetRemoteImage(imageName)
		if err != nil {
			return "", fmt.Errorf("failed to find image %s: %v", imageName, err)
		}
		image = &imageMetadata
	}
	clusterFile, ok := image.Annotations[common.ImageAnnotationForClusterfile]
	if !ok {
		return "", fmt.Errorf("failed to find Clusterfile in local")
	}

	if clusterFile == "" {
		return "", fmt.Errorf("ClusterFile is empty")
	}
	return clusterFile, nil
}

// GetFileFromBaseImage retrieve file from base image
func GetFileFromBaseImage(imageName string, paths ...string) (string, error) {
	mountTarget, _ := utils.MkTmpdir()
	mountUpper, _ := utils.MkTmpdir()
	defer func() {
		utils.CleanDirs(mountTarget, mountUpper)
	}()

	imgSvc, err := NewImageService()
	if err != nil {
		return "", err
	}
	if err := imgSvc.PullIfNotExist(imageName); err != nil {
		return "", err
	}

	driver := mount.NewMountDriver()
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return "", fmt.Errorf("failed to init image store: %s", err)
	}
	image, err := is.GetByName(imageName)
	if err != nil {
		return "", err
	}

	layers, err := GetImageLayerDirs(image)
	if err != nil {
		return "", err
	}

	if err := driver.Mount(mountTarget, mountUpper, layers...); err != nil {
		return "", err
	}

	defer func() {
		if err := driver.Unmount(mountTarget); err != nil {
			logger.Warn(err)
		}
	}()
	var subPath []string
	subPath = append(subPath, mountTarget)
	subPath = append(subPath, paths...)
	clusterFile := filepath.Join(subPath...)

	data, err := ioutil.ReadFile(clusterFile)
	if err != nil {
		return "", err
	}

	if string(data) == "" {
		return "", fmt.Errorf("ClusterFile is empty")
	}

	return string(data), nil
}

func GetYamlByImage(imageName string) (string, error) {
	img, err := GetImageByName(imageName)
	if err != nil {
		return "", fmt.Errorf("failed to get image %s, err: %s", imageName, err)
	}

	ImageInformation, err := yaml.Marshal(img)
	if err != nil {
		return "", err
	}

	return string(ImageInformation), nil
}

func GetImageByName(imageName string) (*v1.Image, error) {
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, fmt.Errorf("failed to init image store, err: %s", err)
	}
	img, err := is.GetByName(imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get image %s, err: %s", imageName, err)
	}
	return img, nil
}
