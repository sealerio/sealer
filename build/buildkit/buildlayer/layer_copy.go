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
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/build/buildkit/buildlayer/imagepuller"

	"github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils"
	"github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils/charts"
	manifest "github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils/manifests"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

// layer handler : implement some ops form the content of each layer
// instruction handler:  implement the execution of instructions

// 1,copy imageList manifests : read the images and fetch them.
// 2,copy nginx.tar registry: load the offline images and push to registry.
// 3,some cmd or run instruction need to do something from the layer content.

type CopyLayer struct {
	Src    string
	Dest   string
	Rootfs string
}

type HandleImageList struct {
	puller imagepuller.Processor
}

func (h HandleImageList) LayerValueHandler(buildContext string, layer v1.Layer) error {
	imageListFilePath := filepath.Join(buildContext, "imageList")
	images, err := h.parseRawImageList(imageListFilePath)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return nil
	}
	// do pull
	return h.puller.Pull(images)
}

func (h HandleImageList) parseRawImageList(imageListFilePath string) ([]string, error) {
	if !utils.IsExist(imageListFilePath) {
		return nil, fmt.Errorf("file imageList is not exist")
	}

	images, err := utils.ReadLines(imageListFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content %s:%v", imageListFilePath, err)
	}
	return FormatImages(images), nil
}

type HandleYamlImageList struct {
	YamlHandler layerutils.Interface
	puller      imagepuller.Processor
	src         string
}

func (h HandleYamlImageList) LayerValueHandler(buildContext string, layer v1.Layer) error {
	yamlFilePath := filepath.Join(buildContext, h.src)
	images, err := h.parseYamlImages(yamlFilePath)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}
	return h.puller.Pull(images)
}

func (h HandleYamlImageList) parseYamlImages(yamlFilePath string) ([]string, error) {
	yamlImages, err := h.YamlHandler.ListImages(yamlFilePath)

	if err != nil {
		return nil, err
	}
	return FormatImages(yamlImages), nil
}

type HandleChartImageList struct {
	ChartHandler layerutils.Interface
	puller       imagepuller.Processor
	src          string
}

func (h HandleChartImageList) LayerValueHandler(buildContext string, layer v1.Layer) error {
	chartFilePath := filepath.Join(buildContext, h.src)
	images, err := h.parseChartImages(chartFilePath)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}
	return h.puller.Pull(images)
}

func (h HandleChartImageList) parseChartImages(chartFilePath string) ([]string, error) {
	chartImages, err := h.ChartHandler.ListImages(chartFilePath)

	if err != nil {
		return nil, err
	}
	return FormatImages(chartImages), nil
}

func NewYamlHandler(lc CopyLayer) *HandleYamlImageList {
	m, _ := manifest.NewManifests()
	return &HandleYamlImageList{
		puller:      imagepuller.NewPuller(lc.Rootfs),
		YamlHandler: m,
		src:         lc.Src,
	}
}

func NewChartHandler(lc CopyLayer) *HandleChartImageList {
	c, _ := charts.NewCharts()
	return &HandleChartImageList{
		puller:       imagepuller.NewPuller(lc.Rootfs),
		src:          lc.Src,
		ChartHandler: c,
	}
}

func NewImageListHandler(lc CopyLayer) *HandleImageList {
	return &HandleImageList{
		puller: imagepuller.NewPuller(lc.Rootfs),
	}
}
