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

package buildlayer

import (
	"strings"

	"github.com/alibaba/sealer/utils"
)

func FormatImages(images []string) (res []string) {
	for _, image := range utils.RemoveDuplicate(images) {
		if image == "" {
			continue
		}
		if strings.HasPrefix(image, "#") {
			continue
		}
		res = append(res, trimQuotes(strings.TrimSpace(image)))
	}
	return
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if c := s[len(s)-1]; s[0] == c && (c == '"' || c == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func GetCopyLayerHandlerType(src, dest string) string {
	//COPY imageList manifests
	//COPY cc charts
	//COPY recommended.yaml manifests
	//COPY nginx.tar images

	if dest == IsCopyToChart {
		return ChartHandler
	}

	if dest == IsCopyOfflineImage {
		return OfflineImageHandler
	}

	if dest == IsCopyToManifests {
		if src == ImageList {
			return ImageListHandler
		}
		if utils.YamlMatcher(src) {
			return YamlHandler
		}
	}

	return ""
}

func ParseCopyLayerContent(layerValue string) (src, dst string) {
	dst = strings.Fields(layerValue)[1]
	for _, p := range []string{"./", "/"} {
		dst = strings.TrimPrefix(dst, p)
	}
	src = strings.Fields(layerValue)[0]
	return
}
