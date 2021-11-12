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

package base

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/config"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/plugin"
	"github.com/alibaba/sealer/runtime"
	"github.com/alibaba/sealer/utils"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

type InitApply struct {
	ImageManager image.Service
	FileSystem   filesystem.Interface
	Runtime      runtime.Interface
	Guest        guest.Interface
	Config       config.Interface
	Plugins      plugin.Plugins
}

func (i InitApply) DoApply(cluster *v1.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster)
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	i.Runtime = runTime
	i.Config = config.NewConfiguration(cluster.Name)
	i.Plugins = plugin.NewPlugins(cluster.Name)

	pipLine, err := i.GetPipeLine()
	if err != nil {
		return err
	}

	for _, f := range pipLine {
		if err = f(cluster); err != nil {
			return err
		}
	}

	return nil
}
func (i InitApply) GetPipeLine() ([]func(cluster *v1.Cluster) error, error) {
	var todoList []func(cluster *v1.Cluster) error
	todoList = append(todoList,
		i.RunInitPlugin,
		i.MountImage,
		i.RunConfig,
		i.MountRootfs,
		i.PluginPhasePreInitRun,
		i.Init,
		i.PluginPhasePreInstallRun,
		i.RunApply,
		i.RunGuest,
		i.UnMountImage,
		i.PluginPhasePostInstallRun,
	)
	return todoList, nil
}
func (i InitApply) RunInitPlugin(cluster *v1.Cluster) error {
	err := i.Plugins.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
	if err != nil {
		return err
	}
	err = i.Plugins.Run(cluster, "Originally")
	if err != nil {
		return err
	}
	return nil
}

func (i InitApply) MountImage(cluster *v1.Cluster) error {
	err := i.ImageManager.PullIfNotExist(cluster.Spec.Image)
	if err != nil {
		return err
	}
	return i.FileSystem.MountImage(cluster)
}

func (i InitApply) RunConfig(cluster *v1.Cluster) error {
	return i.Config.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

func (i InitApply) MountRootfs(cluster *v1.Cluster) error {
	hosts := append(cluster.Spec.Masters.IPList, cluster.Spec.Nodes.IPList...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.Spec.Masters.IPList[0])
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return i.FileSystem.MountRootfs(cluster, hosts, true)
}
func (i InitApply) PluginPhasePreInitRun(cluster *v1.Cluster) error {
	if err := i.Plugins.Load(); err != nil {
		return err
	}
	return i.Plugins.Run(cluster, "PreInit")
}

func (i InitApply) Init(cluster *v1.Cluster) error {
	return i.Runtime.Init(cluster)
}

func (i InitApply) PluginPhasePreInstallRun(cluster *v1.Cluster) error {
	return i.Plugins.Run(cluster, "PreInstall")
}

func (i InitApply) RunApply(cluster *v1.Cluster) error {
	err := i.Runtime.JoinMasters(cluster.Spec.Masters.IPList[1:])
	if err != nil {
		return err
	}
	err = i.Runtime.JoinNodes(cluster.Spec.Nodes.IPList)
	if err != nil {
		return err
	}
	return nil
}

func (i InitApply) RunGuest(cluster *v1.Cluster) error {
	return i.Guest.Apply(cluster)
}
func (i InitApply) UnMountImage(cluster *v1.Cluster) error {
	return i.FileSystem.UnMountImage(cluster)
}

func (i InitApply) PluginPhasePostInstallRun(cluster *v1.Cluster) error {
	return i.Plugins.Run(cluster, "PostInstall")
}

func NewInitApply() (Interface, error) {
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	fs, err := filesystem.NewFilesystem()
	if err != nil {
		return nil, err
	}

	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}

	return InitApply{
		ImageManager: imgSvc,
		FileSystem:   fs,
		Guest:        gs,
	}, nil
}
