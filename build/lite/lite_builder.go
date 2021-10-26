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

	"github.com/alibaba/sealer/build/buildkit/buildimage"
	"github.com/alibaba/sealer/image/reference"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/mount"

	"github.com/alibaba/sealer/build/buildkit/buildinstruction"

	"github.com/alibaba/sealer/build/buildkit"
	"github.com/alibaba/sealer/client/docker"

	"github.com/alibaba/sealer/runtime"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
)

type Builder struct {
	BuildType         string
	NoCache           bool
	ImageNamed        reference.Named
	Context           string
	KubeFileName      string
	DockerClient      *docker.Docker
	RootfsMountTarget *buildinstruction.MountTarget
	BuildImage        buildimage.Interface
}

func (l *Builder) Build(name string, context string, kubefileName string) error {
	named, err := reference.ParseToNamed(name)
	if err != nil {
		return err
	}
	l.ImageNamed = named

	absContext, absKubeFile, err := buildkit.ParseBuildArgs(context, kubefileName)
	if err != nil {
		return err
	}
	l.KubeFileName = absKubeFile

	err = buildkit.ValidateContextDirectory(absContext)
	if err != nil {
		return err
	}
	l.Context = absContext

	bi, err := buildimage.NewBuildImage(absKubeFile)
	if err != nil {
		return err
	}
	l.BuildImage = bi

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
	buildPipeline = append(buildPipeline,
		l.PreCheck,
		l.StartDockerRegistry,
		l.ExecBuild,
		l.SaveBuildImage,
		l.Cleanup,
	)
	return buildPipeline, nil
}

func (l *Builder) PreCheck() error {
	//todo need install sealer docker,if already installed,need to return.
	images, _ := l.DockerClient.ImagesList()
	if len(images) > 0 {
		logger.Warn("The image already exists on the host. Note that the existing image cannot be cached in registry")
	}
	return nil
}

func (l *Builder) mountRootfs() (bool, error) {
	var (
		res = buildinstruction.GetBaseLayersPath(l.BuildImage.GetRawImageBaseLayers())
	)
	// if already mounted ,read mount details set to RootfsMountTarget and return.
	// Negative examples:
	//if pull images failed or exec kubefile instruction failed, run lite build again,will cache part images.
	bindDir := buildkit.GetRegistryBindDir()
	bindTarget := filepath.Dir(bindDir)
	isMounted, upper := mount.GetMountDetails(bindTarget)
	if isMounted {
		logger.Info("get registry cache dir :%s success ", bindTarget)
		registryCache, err := buildinstruction.NewMountTarget(bindTarget, upper, res)
		if err != nil {
			return false, err
		}
		l.RootfsMountTarget = registryCache
		return true, nil
	}

	rootfs, err := buildinstruction.NewMountTarget("", "", res)
	if err != nil {
		return false, err
	}

	err = rootfs.TempMount()
	if err != nil {
		return false, err
	}
	l.RootfsMountTarget = rootfs
	return false, nil
}

func (l *Builder) StartDockerRegistry() error {
	alreadyMount, err := l.mountRootfs()
	if err != nil {
		return err
	}
	if !alreadyMount {
		return l.startRegistry()
	}
	return nil
}

func (l *Builder) startRegistry() error {
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

func (l *Builder) ExecBuild() error {
	ctx := buildimage.Context{
		BuildContext: l.Context,
		BuildType:    l.BuildType,
		UseCache:     !l.NoCache,
	}

	return l.BuildImage.ExecBuild(ctx)
}

func (l *Builder) SaveBuildImage() error {
	layers, err := l.collectLayers()
	if err != nil {
		return err
	}

	imageName := l.ImageNamed.Raw()
	err = l.BuildImage.SaveBuildImage(imageName, layers)
	if err != nil {
		return err
	}
	logger.Info("save image %s to image system success !", imageName)
	return nil
}

func (l *Builder) collectLayers() ([]v1.Layer, error) {
	var layers []v1.Layer
	layers = append(l.BuildImage.GetRawImageBaseLayers(), l.BuildImage.GetRawImageNewLayers()...)
	layer, err := l.collectRegistryCache()
	if err != nil {
		return nil, err
	}

	if layer.ID == "" {
		logger.Warn("registry cache content not found")
		return layers, nil
	}
	layers = append(layers, layer)
	return layers, nil
}

func (l *Builder) collectRegistryCache() (v1.Layer, error) {
	var layer v1.Layer
	upper := l.RootfsMountTarget.GetMountUpper()
	tmp, err := utils.MkTmpdir()
	if err != nil {
		return layer, fmt.Errorf("failed to add upper layer to image, %v", err)
	}
	if utils.IsFileExist(filepath.Join(upper, common.RegistryDirName)) {
		err = os.Rename(filepath.Join(upper, common.RegistryDirName), filepath.Join(tmp, common.RegistryDirName))
		if err != nil {
			return layer, fmt.Errorf("failed to add upper layer to image, %v", err)
		}
	}

	layer, err = l.BuildImage.GenNewLayer(common.BaseImageLayerType, common.RegistryLayerValue, tmp)
	if err != nil {
		return layer, fmt.Errorf("failed to register layer, err: %v", err)
	}

	return layer, nil
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
