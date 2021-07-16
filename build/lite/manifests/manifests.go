package manifest

import (
	"fmt"
	"io/ioutil"
	"os"
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

	err := filepath.Walk(ManifestsRootDir, func(filePath string, fileInfo os.FileInfo, er error) error {
		if er != nil {
			return fmt.Errorf("read file failed %s", er)
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			// skip directories and filename is't .yaml file
			return nil
		}

		yamlBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file failed %s", err)
		}
		images := lite.DecodeImages(string(yamlBytes))
		if len(images) != 0 {
			list = append(list, images...)
		}
		return nil
	})

	if err != nil {
		return list, fmt.Errorf("filepath walk failed %s", err)
	}

	return list, nil
}

func NewManifests() (lite.Interface, error) {
	return &Manifests{}, nil
}

func defaultManifestsRootDir(clusterName string) string {
	return filepath.Join(common.DefaultTheClusterRootfsDir(clusterName), "manifests")
}
