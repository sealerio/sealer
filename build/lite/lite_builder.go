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

package lite

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alibaba/sealer/utils/mount"

	"github.com/alibaba/sealer/build/buildkit"
	"github.com/alibaba/sealer/client/docker"

	"github.com/alibaba/sealer/runtime"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Builder struct {
	Local             *buildkit.Builder
	DockerClient      *docker.Docker
	RootfsMountTarget *buildkit.MountTarget
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

func (l *Builder) GetBuildPipeLine() ([]func() error, error) {
	var buildPipeline []func() error
	if err := l.Local.InitImageSpec(); err != nil {
		return nil, err
	}

	buildPipeline = append(buildPipeline,
		l.PreCheck,
		l.Local.PullBaseImageNotExist,
		l.InitCluster,
		l.MountRootfs,
		l.InitDockerAndRegistry,
		l.Local.ExecBuild,
		l.CollectLiteBuildImages,
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

func (l *Builder) InitCluster() error {
	l.Local.Cluster = &v1.Cluster{}
	l.Local.Cluster.Name = liteBuild
	l.Local.Cluster.Spec.Image = l.Local.Image.Spec.Layers[0].Value
	return nil
}

func (l *Builder) MountRootfs() error {
	err := l.Local.UpdateBuilderLayers(l.Local.Image)
	if err != nil {
		return err
	}
	var (
		target = common.DefaultMountCloudImageDir(l.Local.Cluster.Name)
		upper  = common.DefaultLiteBuildUpper
		res    = buildkit.GetBaseLayersPath(l.Local.BaseLayers)
	)

	if isMount, _ := mount.GetMountDetails(target); isMount {
		err := mount.NewMountDriver().Unmount(target)
		if err != nil {
			return err
		}
	}

	utils.CleanDirs(upper, target)
	err = utils.MkDirs(upper, target)
	if err != nil {
		return err
	}

	rootfs, err := buildkit.NewMountTarget(target, upper, res)
	if err != nil {
		return err
	}

	err = rootfs.TempMount()
	if err != nil {
		return err
	}
	l.RootfsMountTarget = rootfs

	return nil
}

func (l *Builder) InitDockerAndRegistry() error {
	mountedRootfs := l.RootfsMountTarget.GetMountTarget()
	initDockerCmd := fmt.Sprintf("cd %s  && chmod +x scripts/* && cd scripts && bash docker.sh", mountedRootfs)
	host := fmt.Sprintf("%s %s", "127.0.0.1", runtime.SeaHub)
	if !utils.IsFileContent(common.EtcHosts, host) {
		initDockerCmd = fmt.Sprintf("%s && %s", fmt.Sprintf(runtime.RemoteAddEtcHosts, host), initDockerCmd)
	}

	initRegistryCmd := fmt.Sprintf("bash init-registry.sh 5000 %s", filepath.Join(mountedRootfs, common.RegistryDirName))
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

func (l *Builder) CollectLiteBuildImages() error {
	upper := l.RootfsMountTarget.GetMountUpper()
	imageLayer := v1.Layer{
		Type:  "BASE",
		Value: "registry cache",
	}
	tmp, err := utils.MkTmpdir()
	if err != nil {
		return fmt.Errorf("failed to add upper layer to image, %v", err)
	}
	if utils.IsFileExist(filepath.Join(upper, common.RegistryDirName)) {
		err = os.Rename(filepath.Join(upper, common.RegistryDirName), filepath.Join(tmp, common.RegistryDirName))
		if err != nil {
			return fmt.Errorf("failed to add upper layer to image, %v", err)
		}
	}
	layerDgst, err := l.Local.RegisterLayer(tmp)
	if err != nil {
		return err
	}

	imageLayer.ID = layerDgst
	l.Local.NewLayers = append(l.Local.NewLayers, imageLayer)
	return nil
}

func (l *Builder) Cleanup() error {
	if l.RootfsMountTarget != nil {
		l.RootfsMountTarget.CleanUp()
	}

	if err := utils.RemoveFileContent(common.EtcHosts, fmt.Sprintf("127.0.0.1 %s", runtime.SeaHub)); err != nil {
		logger.Warn(err)
	}
	//we need to delete docker registry.if not ,will only cache incremental image in the next lite build
	err := l.DockerClient.RmContainerByName(runtime.RegistryName)
	if err != nil {
		return err
	}

	return utils.CleanFiles(common.RawClusterfile)
}
