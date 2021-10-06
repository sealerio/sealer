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

package manifest

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/sealer/build/lite/liteutils"
	"github.com/alibaba/sealer/common"
)

type Manifests struct{}

// List all the containers images in manifest files
func (manifests *Manifests) ListImages(clusterName string) ([]string, error) {
	var list []string

	ManifestsMountDir := filepath.Join(common.DefaultMountCloudImageDir(clusterName), "manifests")

	err := filepath.Walk(ManifestsMountDir, func(filePath string, fileInfo os.FileInfo, er error) error {
		if er != nil {
			return fmt.Errorf("read file failed %s", er)
		}
		if fileInfo.IsDir() || !strings.HasSuffix(fileInfo.Name(), ".yaml") {
			// skip directories and filename isn't .yaml file
			return nil
		}

		yamlBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file failed %s", err)
		}
		images := liteutils.DecodeImages(string(yamlBytes))
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

func NewManifests() (liteutils.Interface, error) {
	return &Manifests{}, nil
}

func defaultManifestsRootDir(clusterName string) string {
	return filepath.Join(common.DefaultTheClusterRootfsDir(clusterName), "manifests")
}
