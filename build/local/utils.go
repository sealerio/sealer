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

package local

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types/mount"
	"github.com/opencontainers/go-digest"
	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/client/docker"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// GetClusterFile from user build context or from base image
func GetRawClusterFile(im *v1.Image) (string, error) {
	if im.Spec.Layers[0].Value == common.ImageScratch {
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
	if clusterFile, err := getClusterFileFromContext(im); err == nil {
		return clusterFile, nil
	}

	// find cluster file from base image
	return image.GetClusterFileFromImage(im.Spec.Layers[0].Value)
}

func getClusterFileFromContext(image *v1.Image) (string, error) {
	for i := range image.Spec.Layers {
		layer := image.Spec.Layers[i]
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

// used in build stage, where the image still has from layer
func GetBaseLayersPath(layers []v1.Layer) (res []string) {
	for _, layer := range layers {
		if layer.ID != "" {
			res = append(res, filepath.Join(common.DefaultLayerDir, layer.ID.Hex()))
		}
	}
	return res
}

func generateImageID(image v1.Image) (string, error) {
	imageBytes, err := yaml.Marshal(image)
	if err != nil {
		return "", err
	}
	imageID := digest.FromBytes(imageBytes).Hex()
	return imageID, nil
}

func setClusterFileToImage(image *v1.Image, name string) error {
	var cluster v1.Cluster
	clusterFileData, err := GetRawClusterFile(image)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal([]byte(clusterFileData), &cluster); err != nil {
		return err
	}
	cluster.Spec.Image = name
	clusterFile, err := yaml.Marshal(cluster)
	if err != nil {
		return err
	}

	if image.Annotations == nil {
		image.Annotations = make(map[string]string)
	}

	image.Annotations[common.ImageAnnotationForClusterfile] = string(clusterFile)

	return nil
}

func IsOnlyCopy(layers []v1.Layer) bool {
	for i := 1; i < len(layers); i++ {
		if layers[i].Type == common.RUNCOMMAND ||
			layers[i].Type == common.CMDCOMMAND {
			return false
		}
	}
	return true
}

func GetRegistryBindDir() string {
	// check is docker running runtime.RegistryName
	// check bind dir
	var registryName = runtime.RegistryName
	var registryDest = runtime.RegistryBindDest

	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return ""
	}

	containers, err := dockerClient.GetContainerListByName(registryName)

	if err != nil {
		return ""
	}

	for _, c := range containers {
		for _, m := range c.Mounts {
			if m.Type == mount.TypeBind && m.Destination == registryDest {
				return m.Source
			}
		}
	}

	return ""
}

// parse context and kubefile. return context abs path and kubefile abs path
func ParseBuildArgs(localContextDir, kubeFileName string) (string, string, error) {
	localDir, err := resolveAndValidateContextPath(localContextDir)
	if err != nil {
		return "", "", err
	}

	if kubeFileName != "" {
		if kubeFileName, err = filepath.Abs(kubeFileName); err != nil {
			return "", "", fmt.Errorf("unable to get absolute path to KubeFile: %v", err)
		}
	}

	relFileName, err := getKubeFileRelPath(localDir, kubeFileName)
	return localDir, relFileName, err
}

func resolveAndValidateContextPath(givenContextDir string) (string, error) {
	absContextDir, err := filepath.Abs(givenContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to get absolute context directory %s: %v", givenContextDir, err)
	}

	absContextDir, err = filepath.EvalSymlinks(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in context path: %v", err)
	}

	stat, err := os.Lstat(absContextDir)
	if err != nil {
		return "", fmt.Errorf("unable to stat context directory %s: %v", absContextDir, err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("context must be a directory: %s", absContextDir)
	}

	return absContextDir, err
}

func getKubeFileRelPath(absContextDir, givenKubeFile string) (string, error) {
	var err error

	absKubeFile := givenKubeFile
	if absKubeFile == "" {
		absKubeFile = filepath.Join(absContextDir, kubefile)
		if _, err = os.Lstat(absKubeFile); os.IsNotExist(err) {
			altPath := filepath.Join(absContextDir, strings.ToLower(kubefile))
			if _, err = os.Lstat(altPath); err == nil {
				absKubeFile = altPath
			}
		}
	}

	if !filepath.IsAbs(absKubeFile) {
		absKubeFile = filepath.Join(absContextDir, absKubeFile)
	}

	absKubeFile, err = filepath.EvalSymlinks(absKubeFile)
	if err != nil {
		return "", fmt.Errorf("unable to evaluate symlinks in KubeFile path: %v", err)
	}

	if _, err := os.Lstat(absKubeFile); err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("cannot locate KubeFile: %s", absKubeFile)
		}
		return "", fmt.Errorf("unable to stat KubeFile: %v", err)
	}

	return absKubeFile, nil
}

func ValidateContextDirectory(srcPath string) error {
	contextRoot, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}

	return filepath.Walk(contextRoot, func(filePath string, f os.FileInfo, err error) error {
		if err != nil {
			if os.IsPermission(err) {
				return fmt.Errorf("can't stat '%s'", filePath)
			}
			if os.IsNotExist(err) {
				return fmt.Errorf("file '%s' not found", filePath)
			}
			return err
		}

		if f.IsDir() {
			return nil
		}

		if f.Mode()&(os.ModeSymlink|os.ModeNamedPipe) != 0 {
			return nil
		}

		currentFile, err := os.Open(filePath)
		if err != nil && os.IsPermission(err) {
			return fmt.Errorf("no permission to read from '%s'", filePath)
		}
		currentFile.Close()

		return nil
	})
}
