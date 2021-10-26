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

package buildimage

import (
	"fmt"
	"io/ioutil"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/parser"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"github.com/opencontainers/go-digest"

	"path/filepath"
	"strings"

	"sigs.k8s.io/yaml"
)

// InitImageSpec init default Image metadata
func InitImageSpec(kubefile string) (*v1.Image, error) {
	kubeFile, err := utils.ReadAll(kubefile)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubefile: %v", err)
	}

	rawImage := parser.NewParse().Parse(kubeFile)
	if rawImage == nil {
		return nil, fmt.Errorf("failed to parse kubefile, image is nil")
	}

	layer0 := rawImage.Spec.Layers[0]
	if layer0.Type != common.FROMCOMMAND {
		return nil, fmt.Errorf("first line of kubefile must start with %s", common.FROMCOMMAND)
	}

	return rawImage, nil
}

func setClusterFileToImage(clusterFile string, newImage string, image *v1.Image) error {
	var cluster v1.Cluster
	if err := yaml.Unmarshal([]byte(clusterFile), &cluster); err != nil {
		return err
	}
	cluster.Spec.Image = newImage
	clusterData, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	if image.Annotations == nil {
		image.Annotations = make(map[string]string)
	}
	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterData)
	return nil
}

// GetRawClusterFile GetClusterFile from user build context or from base image
func GetRawClusterFile(baseImage string, layers []v1.Layer) (string, error) {
	if baseImage == common.ImageScratch {
		data, err := ioutil.ReadFile(filepath.Join("etc", common.DefaultClusterFileName))
		if err != nil {
			return "", err
		}
		if string(data) == "" {
			return "", fmt.Errorf("ClusterFile content is empty")
		}
		return string(data), nil
	}

	// find cluster file from context
	if clusterFile, err := getClusterFileFromContext(layers); err == nil {
		return clusterFile, nil
	}

	// find cluster file from base image
	return image.GetClusterFileFromImage(baseImage)
}

func getClusterFileFromContext(layers []v1.Layer) (string, error) {
	for i := range layers {
		layer := layers[i]
		if layer.Type == common.COPYCOMMAND && strings.Fields(layer.Value)[0] == common.DefaultClusterFileName {
			clusterFile, err := utils.ReadAll(strings.Fields(layer.Value)[0])
			if err != nil {
				return "", err
			}
			if string(clusterFile) == "" {
				return "", fmt.Errorf("ClusterFile is empty")
			}
			return string(clusterFile), nil
		}
	}
	return "", fmt.Errorf("failed to get ClusterFile from Context")
}

func generateImageID(image v1.Image) (string, error) {
	imageBytes, err := yaml.Marshal(image)
	if err != nil {
		return "", err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	return imageID, nil
}
