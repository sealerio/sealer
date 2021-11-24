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

	"github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils"
	"github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils/charts"
	manifest "github.com/alibaba/sealer/build/buildkit/buildlayer/layerutils/manifests"
	"github.com/alibaba/sealer/client/docker"
	"github.com/alibaba/sealer/utils"
)

// layer handler : implement some ops form the content of each layer
// instruction handler:  implement the execution of instructions

// 1,copy imageList manifests : read the images and fetch them.
// 2,copy nginx.tar registry: load the offline images and push to registry.
// 3,some cmd or run instruction need to do something from the layer content.

type LayerCopy struct {
	Src         string
	Dest        string
	HandlerType string
}

//if support multiple container runtime(such as container) when build,should change element "DockerClient" to
//a interface.involving three structs:HandleImageList,HandleYamlImageList,HandleChartImageList.
type HandleImageList struct {
	DockerClient *docker.Docker
	Lc           LayerCopy
}

func (h HandleImageList) LayerValueHandler(buildContext string, SealerDocker bool) error {
	imageListFilePath := filepath.Join(buildContext, "imageList")
	images, err := h.parseRawImageList(imageListFilePath)
	if err != nil {
		return err
	}
	return h.DockerClient.ImagesCacheToRegistry(images, SealerDocker)
}

func (h HandleImageList) parseRawImageList(imageListFilePath string) ([]string, error) {
	if !utils.IsExist(imageListFilePath) {
		return nil, fmt.Errorf("file imageList is not exist")
	}

	images, err := utils.ReadLines(imageListFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content %s:%v", imageListFilePath, err)
	}
	return images, nil
}

type HandleYamlImageList struct {
	DockerClient *docker.Docker
	YamlHandler  layerutils.Interface
	Lc           LayerCopy
}

func (h HandleYamlImageList) LayerValueHandler(buildContext string, SealerDocker bool) error {
	yamlFilePath := filepath.Join(buildContext, h.Lc.Src)
	images, err := h.parseYamlImages(yamlFilePath)
	if err != nil {
		return err
	}
	return h.DockerClient.ImagesCacheToRegistry(images, SealerDocker)
}

func (h HandleYamlImageList) parseYamlImages(yamlFilePath string) ([]string, error) {
	yamlImages, err := h.YamlHandler.ListImages(yamlFilePath)

	if err != nil {
		return nil, err
	}

	return yamlImages, nil
}

type HandleChartImageList struct {
	DockerClient *docker.Docker
	ChartHandler layerutils.Interface
	Lc           LayerCopy
}

func (h HandleChartImageList) LayerValueHandler(buildContext string, SealerDocker bool) error {
	chartFilePath := filepath.Join(buildContext, h.Lc.Src)
	images, err := h.parseChartImages(chartFilePath)
	if err != nil {
		return err
	}
	return h.DockerClient.ImagesCacheToRegistry(images, SealerDocker)
}

func (h HandleChartImageList) parseChartImages(chartFilePath string) ([]string, error) {
	chartImages, err := h.ChartHandler.ListImages(chartFilePath)

	if err != nil {
		return nil, err
	}

	return chartImages, nil
}

func NewYamlHandler(lc LayerCopy) *HandleYamlImageList {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil
	}
	m, _ := manifest.NewManifests()

	return &HandleYamlImageList{
		DockerClient: dockerClient,
		Lc:           lc,
		YamlHandler:  m,
	}
}

func NewChartHandler(lc LayerCopy) *HandleChartImageList {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil
	}
	c, _ := charts.NewCharts()

	return &HandleChartImageList{
		DockerClient: dockerClient,
		Lc:           lc,
		ChartHandler: c,
	}
}

func NewImageListHandler(lc LayerCopy) *HandleImageList {
	dockerClient, err := docker.NewDockerClient()
	if err != nil {
		return nil
	}

	return &HandleImageList{
		DockerClient: dockerClient,
		Lc:           lc,
	}
}
