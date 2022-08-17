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

	"github.com/sealerio/sealer/build/layerutils"
)

type Charts struct{}

// ListImages List all the containers images in helm charts
// TODO Current logic will lead to leave the unreserved image
// The image will be ignored if it has been written in the yaml(image: docker.io/xxx)
func (charts *Charts) ListImages(chartPath string) ([]string, error) {
	var list []string
	images, err := GetImageList(chartPath)
	if err != nil {
		return list, fmt.Errorf("failed to get images chart path(%s), err: %s", chartPath, err)
	}
	if len(images) != 0 {
		list = append(list, images...)
	}
	return list, nil
}

func NewCharts() (layerutils.Interface, error) {
	return &Charts{}, nil
}
