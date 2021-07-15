package manifest

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/build/lite"
	"github.com/alibaba/sealer/common"
)

type Manifests struct{}

// List all the containers images in manifest files
func (manifests *Manifests) ListImages(clusterName string) ([]string, error) {
	var list []string

	ManifestsRootDir := defaultManifestsRootDir(clusterName)
	files, err := ioutil.ReadDir(ManifestsRootDir)
	if err != nil {
		return list, fmt.Errorf("list images failed %s", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".yaml") {
			// skip directories and filename is't .yaml file
			continue
		}
		manifestFilePath := filepath.Join(ManifestsRootDir, file.Name())
		yamlBytes, err := ioutil.ReadFile(manifestFilePath)
		if err != nil {
			return list, fmt.Errorf("read file failed %s", err)
		}
		images := lite.DecodeImages(string(yamlBytes))
		if len(images) != 0 {
			list = append(list, images...)
		}
	}

	return list, nil
}

func NewManifests() (lite.Interface, error) {
	return &Manifests{}, nil
}

func defaultManifestsRootDir(clusterName string) string {
	return filepath.Join(common.DefaultTheClusterRootfsDir(clusterName), "manifests")
}
