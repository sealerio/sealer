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
	"path/filepath"

	"github.com/alibaba/sealer/utils/mount"

	"github.com/alibaba/sealer/runtime"

	"github.com/alibaba/sealer/build/lite/charts"
	"github.com/alibaba/sealer/build/lite/docker"
	manifest "github.com/alibaba/sealer/build/lite/manifests"
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type LiteBuilder struct {
	local *LocalBuilder
}

const liteBuild = "lite-build"

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

// load cluster file from disk
func (l *LiteBuilder) InitCluster() error {
	l.local.Cluster = &v1.Cluster{}
	l.local.Cluster.Name = liteBuild
	l.local.Cluster.Spec.Image = l.local.Image.Spec.Layers[0].Value
	return nil
}

func (l *LiteBuilder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.local.InitImageSpec(); err != nil {
		return nil, err
	}

	buildPipeline = append(buildPipeline,
		l.PreCheck,
		l.local.PullBaseImageNotExist,
		l.InitCluster,
		l.local.ExecBuild,
		l.MountImage,
		l.InitDockerAndRegistry,
		l.CacheImageToRegistry,
		l.AddUpperLayerToImage,
		l.Clear,
	)
	return buildPipeline, nil
}

func (l *LiteBuilder) PreCheck() error {
	d := docker.Docker{}
	images, _ := d.ImagesList()
	if len(images) > 0 {
		logger.Warn("The image already exists on the host. Note that the existing image cannot be cached in registry")
	}
	return nil
}

func (l *LiteBuilder) MountImage() error {
	if isMount, _ := mount.GetMountDetails(common.DefaultMountCloudImageDir(l.local.Cluster.Name)); isMount {
		err := mount.NewMountDriver().Unmount(common.DefaultMountCloudImageDir(l.local.Cluster.Name))
		if err != nil {
			return err
		}
	}
	res := getBaseLayersPath(append(l.local.baseLayers, l.local.newLayers...))
	upper := common.DefaultLiteBuildUpper
	utils.CleanDir(upper)
	err := utils.MkDirs(upper, common.DefaultMountCloudImageDir(l.local.Cluster.Name))
	if err != nil {
		return err
	}
	return mount.NewMountDriver().Mount(common.DefaultMountCloudImageDir(l.local.Cluster.Name), upper, res...)
}

func (l *LiteBuilder) AddUpperLayerToImage() error {
	upper := common.DefaultLiteBuildUpper
	imageLayer := v1.Layer{
		Type:  "BASE",
		Value: "registry cache",
	}
	utils.CleanDirs(filepath.Join(upper, "scripts"), filepath.Join(upper, "cri"))
	layerDgst, err := l.local.registerLayer(upper)
	if err != nil {
		return err
	}

	imageLayer.ID = layerDgst
	l.local.newLayers = append(l.local.newLayers, imageLayer)
	err = l.local.UpdateImageMetadata()
	if err != nil {
		return err
	}
	return nil
}

func (l *LiteBuilder) Clear() error {
	mountDir := common.DefaultMountCloudImageDir(l.local.Cluster.Name)
	err := mount.NewMountDriver().Unmount(mountDir)
	if err != nil {
		return fmt.Errorf("failed to umount %s, %v", mountDir, err)
	}
	if err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("127.0.0.1 %s", runtime.SeaHub)); err != nil {
		logger.Warn(err)
	}
	return utils.CleanFiles(common.RawClusterfile, common.DefaultClusterBaseDir(l.local.Cluster.Name), filepath.Join(common.DefaultTmpDir, common.DefaultLiteBuildUpper))
}

func (l *LiteBuilder) InitDockerAndRegistry() error {
	mount := filepath.Join(common.DefaultClusterBaseDir(l.local.Cluster.Name), "mount")
	initDockerCmd := fmt.Sprintf("cd %s  && chmod +x scripts/* && cd scripts && bash docker.sh", mount)
	host := fmt.Sprintf("%s %s", "127.0.0.1", runtime.SeaHub)
	if !utils.IsFileContent(common.EtcHosts, host) {
		initDockerCmd = fmt.Sprintf("%s && %s", fmt.Sprintf(runtime.RemoteAddEtcHosts, host), initDockerCmd)
	}
	initRegistryCmd := fmt.Sprintf("bash init-registry.sh 5000 %s", filepath.Join(mount, "registry"))
	r, err := utils.RunSimpleCmd(fmt.Sprintf("%s && %s", initDockerCmd, initRegistryCmd))
	logger.Info(r)
	if err != nil {
		logger.Error(fmt.Sprintf("Init docker and registry failed: %v", err))
		return err
	}
	return nil
}

func (l *LiteBuilder) CacheImageToRegistry() error {
	var images []string
	var err error
	d := docker.Docker{}
	c := charts.Charts{}
	m := manifest.Manifests{}
	imageList := filepath.Join(common.DefaultClusterBaseDir(l.local.Cluster.Name), "mount", "manifests", "imageList")
	if utils.IsExist(imageList) {
		images, err = utils.ReadLines(imageList)
	}
	if helmImages, err := c.ListImages(l.local.Cluster.Name); err == nil {
		images = append(images, helmImages...)
	}
	if manifestImages, err := m.ListImages(l.local.Cluster.Name); err == nil {
		images = append(images, manifestImages...)
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
