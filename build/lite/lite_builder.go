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
	"time"

	"github.com/alibaba/sealer/build/lite/liteutils/charts"
	manifest "github.com/alibaba/sealer/build/lite/liteutils/manifests"
	"github.com/alibaba/sealer/build/local"
	"github.com/alibaba/sealer/client/docker"

	"github.com/alibaba/sealer/utils/mount"

	"github.com/alibaba/sealer/runtime"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Builder struct {
	Local        *local.Builder
	DockerClient *docker.Docker
}

const liteBuild = "lite-build"

func (l *Builder) Build(name string, context string, kubefileName string) error {
	err := l.Local.InitBuilder(name, context, kubefileName)
	if err != nil {
		return err
	}

	l.DockerClient, err = docker.NewDockerClient()
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
func (l *Builder) InitCluster() error {
	l.Local.Cluster = &v1.Cluster{}
	l.Local.Cluster.Name = liteBuild
	l.Local.Cluster.Spec.Image = l.Local.Image.Spec.Layers[0].Value
	return nil
}

func (l *Builder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.Local.InitImageSpec(); err != nil {
		return nil, err
	}

	buildPipeline = append(buildPipeline,
		l.PreCheck,
		l.Local.PullBaseImageNotExist,
		l.InitCluster,
		l.Local.ExecBuild,
		l.MountImage,
		l.InitDockerAndRegistry,
		l.CacheImageToRegistry,
		l.AddUpperLayerToImage,
		l.Local.UpdateImageMetadata,
		l.Cleanup,
	)
	return buildPipeline, nil
}

func (l *Builder) PreCheck() error {
	images, _ := l.DockerClient.ImagesList()
	if len(images) > 0 {
		logger.Warn("The image already exists on the host. Note that the existing image cannot be cached in registry")
	}
	return nil
}

func (l *Builder) MountImage() error {
	if isMount, _ := mount.GetMountDetails(common.DefaultMountCloudImageDir(l.Local.Cluster.Name)); isMount {
		err := mount.NewMountDriver().Unmount(common.DefaultMountCloudImageDir(l.Local.Cluster.Name))
		if err != nil {
			return err
		}
	}
	res := local.GetBaseLayersPath(append(l.Local.BaseLayers, l.Local.NewLayers...))
	upper := common.DefaultLiteBuildUpper
	utils.CleanDirs(upper, common.DefaultMountCloudImageDir(l.Local.Cluster.Name))
	err := utils.MkDirs(upper, common.DefaultMountCloudImageDir(l.Local.Cluster.Name))
	if err != nil {
		return err
	}
	return mount.NewMountDriver().Mount(common.DefaultMountCloudImageDir(l.Local.Cluster.Name), upper, res...)
}

func (l *Builder) AddUpperLayerToImage() error {
	upper := common.DefaultLiteBuildUpper
	imageLayer := v1.Layer{
		Type:  "BASE",
		Value: "registry cache",
	}
	layerDgst, err := l.local.registerLayer(upper)
	if err != nil {
		return fmt.Errorf("failed to register layer, err: %v", err)
	}

	imageLayer.ID = layerDgst
	l.Local.NewLayers = append(l.Local.NewLayers, imageLayer)
	return nil
}

func (l *Builder) Cleanup() error {
	mountDir := common.DefaultMountCloudImageDir(l.Local.Cluster.Name)
	err := mount.NewMountDriver().Unmount(mountDir)
	if err != nil {
		return fmt.Errorf("failed to umount %s, %v", mountDir, err)
	}
	if err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("127.0.0.1 %s", runtime.SeaHub)); err != nil {
		logger.Warn(err)
	}
	//we need to delete docker registry.if not ,will only cache incremental image in the next lite build
	err = l.DockerClient.RmContainerByName(runtime.RegistryName)
	if err != nil {
		return err
	}

	return utils.CleanFiles(common.RawClusterfile, common.DefaultClusterBaseDir(l.Local.Cluster.Name), filepath.Join(common.DefaultTmpDir, common.DefaultLiteBuildUpper))
}

func (l *Builder) InitDockerAndRegistry() error {
	mountedRootfs := filepath.Join(common.DefaultClusterBaseDir(l.Local.Cluster.Name), "mount")
	initDockerCmd := fmt.Sprintf("cd %s  && chmod +x scripts/* && cd scripts && bash docker.sh", mountedRootfs)
	host := fmt.Sprintf("%s %s", "127.0.0.1", runtime.SeaHub)
	if !utils.IsFileContent(common.EtcHosts, host) {
		initDockerCmd = fmt.Sprintf("%s && %s", fmt.Sprintf(runtime.RemoteAddEtcHosts, host), initDockerCmd)
	}
	initRegistryCmd := fmt.Sprintf("bash init-registry.sh 5000 %s", filepath.Join(mountedRootfs, "registry"))
	r, err := utils.RunSimpleCmd(fmt.Sprintf("%s && %s", initDockerCmd, initRegistryCmd))
	logger.Info(r)
	if err != nil {
		return fmt.Errorf("failed to init docker and registry: %v", err)
	}
	return utils.Retry(10, 3*time.Second, func() error {
		if !utils.IsHostPortExist("tcp", "127.0.0.1", 5000) {
			return fmt.Errorf("registry is not ready")
		}
		return nil
	})
}

func (l *Builder) CacheImageToRegistry() error {
	var images []string
	var err error
	c, _ := charts.NewCharts()
	m, _ := manifest.NewManifests()
	imageList := filepath.Join(common.DefaultClusterBaseDir(l.Local.Cluster.Name), "mount", "manifests", "imageList")
	if utils.IsExist(imageList) {
		images, err = utils.ReadLines(imageList)
	}
	if helmImages, err := c.ListImages(l.Local.Cluster.Name); err == nil {
		images = append(images, helmImages...)
	}
	if manifestImages, err := m.ListImages(l.Local.Cluster.Name); err == nil {
		images = append(images, manifestImages...)
	}
	if err != nil {
		return err
	}
	l.DockerClient.ImagesPull(images)
	return nil
}
