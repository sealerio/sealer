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
	"path/filepath"
	"strings"

	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/image/store"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// GetImageLayerDirs return image hash list
// current image is different with the image in build stage
// current image hasn't from layer
func GetImageLayerDirs(image *v1.Image) (res []string, err error) {
	for _, layer := range image.Spec.Layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return
}

func GetImageDetails(idOrName string, platforms []*v1.Platform) (string, error) {
	var isImageID bool
	var imgs []*v1.Image

	if idOrName == "" {
		return "", fmt.Errorf("image id or name cannot be empty")
	}
	imageStore, err := store.NewDefaultImageStore()
	if err != nil {
		return "", fmt.Errorf("failed to init image store, err: %s", err)
	}
	imageMetadataMap, err := imageStore.GetImageMetadataMap()
	if err != nil {
		return "", err
	}

	// detect if the input is image id.
	for _, manifestList := range imageMetadataMap {
		for _, m := range manifestList.Manifests {
			if m.ID == idOrName {
				isImageID = true
				break
			}
		}
	}

	if isImageID {
		ima, err := imageStore.GetByID(idOrName)
		if err != nil {
			return "", err
		}
		imgs = append(imgs, ima)
	} else {
		ima, err := getImageByName(idOrName, platforms, imageStore)
		if err != nil {
			return "", err
		}
		imgs = append(imgs, ima...)
	}

	info, err := yaml.Marshal(imgs)
	if err != nil {
		return "", err
	}

	return string(info), nil
}

func getImageByName(imageName string, platforms []*v1.Platform, is store.ImageStore) ([]*v1.Image, error) {
	var imgs []*v1.Image

	imageManifests, err := is.GetImageManifestList(imageName)
	if err != nil {
		return nil, err
	}

	if len(platforms) == 0 {
		for _, m := range imageManifests {
			ima, err := is.GetByID(m.ID)
			if err != nil {
				return nil, err
			}
			imgs = append(imgs, ima)
		}
		return imgs, nil
	}

	for _, p := range platforms {
		ima, err := is.GetByName(imageName, p)
		if err != nil {
			return nil, fmt.Errorf("failed to get image %s, err: %s", imageName, err)
		}
		imgs = append(imgs, ima)
	}
	return imgs, nil
}

func setClusterFile(imageName string, image *v1.Image) error {
	var cluster v2.Cluster
	if image.Annotations == nil {
		return nil
	}
	raw := image.Annotations[common.ImageAnnotationForClusterfile]
	if err := yaml.Unmarshal([]byte(raw), &cluster); err != nil {
		return err
	}
	cluster.Spec.Image = imageName
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterData)
	return nil
}

func GenerateImageID(image v1.Image) (string, error) {
	imageBytes, err := yaml.Marshal(image)
	if err != nil {
		return "", err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	return imageID, nil
}

func ConvertToHostname(url string) string {
	stripped := url
	if strings.HasPrefix(url, "http://") {
		stripped = strings.TrimPrefix(url, "http://")
	} else if strings.HasPrefix(url, "https://") {
		stripped = strings.TrimPrefix(url, "https://")
	}

	nameParts := strings.SplitN(stripped, "/", 2)

	return nameParts[0]
}
