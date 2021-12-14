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
	ocispecs "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

type ParseLayerOpts struct {
	ImageSaveDir string
	Platform     ocispecs.Platform
}

func NewParseLayerOpts(saveDir string, platform ocispecs.Platform) *ParseLayerOpts {
	return &ParseLayerOpts{
		ImageSaveDir: saveDir,
		Platform:     platform,
	}
}

// ParseLayerContent :init different layer handler to exchanging due to the layer content
func ParseLayerContent(layer *v1.Layer, opts *ParseLayerOpts) LayerHandler {
	var layerParser CopyLayer
	if layer.Type == common.COPYCOMMAND {
		// parse copy attr
		layerParser = ParseCopyLayerValue(layer.Value)
		// parse image attr
		layerParser.ImageSaveDir = opts.ImageSaveDir
		layerParser.Platform = opts.Platform
	}

	switch layerParser.HandlerType {
	// imageList;yaml,chart
	case ImageListHandler:
		return NewImageListHandler(layerParser)
	case YamlHandler:
		return NewYamlHandler(layerParser)
	case ChartHandler:
		return NewChartHandler(layerParser)
	}
	return nil
}

func ParseCopyLayerValue(layerValue string) CopyLayer {
	//COPY imageList manifests
	//COPY cc charts
	//COPY recommended.yaml manifests
	//COPY nginx.tar images

	dst := strings.Fields(layerValue)[1]
	for _, p := range []string{"./", "/"} {
		dst = strings.TrimPrefix(dst, p)
	}

	lc := CopyLayer{
		Src:  strings.Fields(layerValue)[0],
		Dest: dst,
	}
	if lc.Dest == IsCopyToManifests {
		if lc.Src == ImageList {
			lc.HandlerType = ImageListHandler
		}
		if utils.YamlMatcher(lc.Src) {
			lc.HandlerType = YamlHandler
		}
		return lc
	}

	if lc.Dest == IsCopyToChart {
		lc.HandlerType = ChartHandler
		return lc
	}

	if lc.Dest == IsCopyOfflineImage {
		lc.HandlerType = OfflineImageHandler
		return lc
	}

	return lc
}
