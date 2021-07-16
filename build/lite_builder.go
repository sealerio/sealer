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

package build

import (
	"fmt"
	"github.com/alibaba/sealer/build/lite/charts"
	"github.com/alibaba/sealer/build/lite/docker"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"path/filepath"
)

type LiteBuilder struct {
	local *LocalBuilder
}

func (l *LiteBuilder) Build(name string, context string, kubefileName string) error {
	err := l.local.initBuilder(name, context, kubefileName)
	if err != nil {
		return err
	}

	pipLine, err := l.GetBuildPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(); err != nil {
			return err
		}
	}
	return nil
}

func (l *LiteBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.local.InitImageSpec(); err != nil {
		return nil, err
	}

	buildPipeline = append(buildPipeline,
		l.local.PullBaseImageNotExist,
		l.MountImage,
		l.InitDockerAndRegistry,
		l.CacheImageToRegistry,
		l.local.ExecBuild,
		l.local.UpdateImageMetadata)
	return buildPipeline, nil
}

func (l *LiteBuilder) MountImage() error {
	FileSystem := filesystem.NewFilesystem()
	if err := FileSystem.MountImage(l.local.Cluster); err != nil {
		return err
	}
	return nil
}

func (l *LiteBuilder) InitDockerAndRegistry() error {
	rootfs := common.DefaultClusterBaseDir(l.local.Cluster.Name)
	cmd := "cd %s  && chmod +x scripts/* && cd scripts && sh docker.sh && sh init-registry.sh 5000 %s"
	r, err := utils.CmdOutput(fmt.Sprintf(cmd, rootfs, filepath.Join(rootfs, "registry")))
	if err != nil {
		logger.Error(fmt.Sprintf("Init docker and registry failed: %v", err))
		return err
	}
	logger.Info(string(r))
	return nil
}

func (l *LiteBuilder) CacheImageToRegistry() error {
	var images []string
	var err error
	d := docker.Docker{}
	c := charts.Charts{}
	imageList := filepath.Join(common.DefaultClusterBaseDir(l.local.Cluster.Name), "imageList")
	if utils.IsExist(imageList) {
		images, err = utils.ReadLines(imageList)
	}
	if i, err := c.ListImages(l.local.Cluster.Name); err == nil {
		images = append(images, i...)
	}
	if err != nil {
		return err
	}
	d.ImagesPull(images)
	return nil
}

func NewLiteBuilder(config *Config) (Interface, error) {
	localBuilder, err := NewLocalBuilder(config)
	if err != nil {
		return nil, err
	}
	return &LiteBuilder{
		local: localBuilder.(*LocalBuilder),
	}, nil
}
