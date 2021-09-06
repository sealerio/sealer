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

package charts

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
)

type Charts struct{}

// List all the containers images in helm charts
func (charts *Charts) ListImages(clusterName string) ([]string, error) {
	var list []string

	chartsRootDir := defaultChartsRootDir(clusterName)
	files, err := ioutil.ReadDir(chartsRootDir)
	if err != nil {
		return list, fmt.Errorf("list images failed %s", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			// skip files
			continue
		}
		chartPath := filepath.Join(chartsRootDir, file.Name())
		images, err := GetImageList(chartPath)
		if err != nil {
			return list, fmt.Errorf("get images failed,chart path:%s, err: %s", chartPath, err)
		}
		if len(images) != 0 {
			list = append(list, images...)
		}
	}

	list = removeDuplicate(list)
	return list, nil
}

func NewCharts() (Interface, error) {
	return &Charts{}, nil
}

func defaultChartsRootDir(clusterName string) string {
	return filepath.Join(common.DefaultTheClusterRootfsDir(clusterName), "charts")
}

func removeDuplicate(images []string) []string {
	var result []string
	flagMap := map[string]struct{}{}

	for _, image := range images {
		if _, ok := flagMap[image]; !ok {
			flagMap[image] = struct{}{}
			result = append(result, image)
		}
	}
	return result
}
