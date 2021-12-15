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
	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

// ParseLayerContent :init different layer handler to exchanging due to the layer content
func ParseLayerContent(rootfs string, layer *v1.Layer) LayerHandler {
	if layer.Type != common.COPYCOMMAND {
		return nil
	}
	src, dest := ParseCopyLayerContent(layer.Value)
	// parse copy attr
	ht := GetCopyLayerHandlerType(src, dest)
	if ht == "" {
		return nil
	}

	cl := CopyLayer{
		Src:    src,
		Dest:   dest,
		Rootfs: rootfs,
	}

	switch ht {
	// imageList;yaml,chart
	case ImageListHandler:
		return NewImageListHandler(cl)
	case YamlHandler:
		return NewYamlHandler(cl)
	case ChartHandler:
		return NewChartHandler(cl)
	}
	return nil
}
