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

package applyentity

import (
	"fmt"

	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/pkg/config"
	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/guest"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/plugin"
	"github.com/alibaba/sealer/utils"
)

type InitApply struct {
	ImageManager image.Service
	FileSystem   filesystem.Interface
	Runtime      runtime.Interface
	Guest        guest.Interface
	Config       config.Interface
	Plugins      plugin.Plugins
}

func (i InitApply) DoApply(cluster *v2.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster, cluster.GetAnnotationsByKey(common.ClusterfileName))
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
func (i InitApply) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		//i.RunInitPlugin,
		i.MountImage,
		i.RunConfig,
		i.MountRootfs,
		//i.PluginPhasePreInitRun,
		i.Init,
		i.RunApply,
		//i.PluginPhasePreGuestRun,
		i.RunGuest,
		i.UnMountImage,
		//i.PluginPhasePostInstallRun,
	)
	return todoList, nil
}

func (i InitApply) MountImage(cluster *v2.Cluster) error {
	err := i.ImageManager.PullIfNotExist(cluster.Spec.Image)
	if err != nil {
		return err
	}
	return i.FileSystem.MountImage(cluster)
}

func (i InitApply) RunConfig(cluster *v2.Cluster) error {
	return i.Config.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

func (i InitApply) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0Ip())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return i.FileSystem.MountRootfs(cluster, hosts, true)
}

func (i InitApply) Init(cluster *v2.Cluster) error {
	return i.Runtime.Init(cluster)
}

func (i InitApply) RunApply(cluster *v2.Cluster) error {
	err := i.Runtime.JoinMasters(cluster.GetMasterIPList()[1:])
	if err != nil {
		return err
	}
	err = i.Runtime.JoinNodes(cluster.GetNodeIPList())
	if err != nil {
		return err
	}
	return nil
}

func (i InitApply) RunGuest(cluster *v2.Cluster) error {
	return i.Guest.Apply(cluster)
}
func (i InitApply) UnMountImage(cluster *v2.Cluster) error {
	return i.FileSystem.UnMountImage(cluster)
}

/*func (i InitApply) RunInitPlugin(cluster *v2.Cluster) error {
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

func (i InitApply) PluginPhasePreInitRun(cluster *v2.Cluster) error {
	if err := i.Plugins.Load(); err != nil {
		return err
	}
	return i.Plugins.Run(cluster, "PreInit")
}

func (i InitApply) PluginPhasePreGuestRun(cluster *v2.Cluster) error {
	return i.Plugins.Run(cluster, "PreGuest")
}

func (i InitApply) PluginPhasePostInstallRun(cluster *v2.Cluster) error {
	return i.Plugins.Run(cluster, "PostInstall")
}*/

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
