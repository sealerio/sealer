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

package processor

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

type Creation struct {
	ImageManager image.Service
	FileSystem   filesystem.Interface
	Runtime      runtime.Interface
	Guest        guest.Interface
	Config       config.Interface
	Plugins      plugin.Plugins
}

//var pluginPhases = []plugin.Phase{plugin.PhaseOriginally, plugin.PhasePreInit, plugin.PhasePreGuest, plugin.PhasePostInstall}

func (c Creation) Execute(cluster *v2.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster, cluster.GetAnnotationsByKey(common.ClusterfileName))
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	c.Runtime = runTime
	c.Config = config.NewConfiguration(cluster.Name)
	if err := c.initPlugin(cluster); err != nil {
		return err
	}

	pipLine, err := c.GetPipeLine()
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
func (c Creation) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		//c.RunInitPlugin,
		c.MountImage,
		c.RunConfig,
		c.MountRootfs,
		//c.PluginPhasePreInitRun,
		c.Init,
		c.Join,
		//c.PluginPhasePreGuestRun,
		c.RunGuest,
		c.UnMountImage,
		//c.PluginPhasePostInstallRun,
	)
	return todoList, nil
}

func (c Creation) MountImage(cluster *v2.Cluster) error {
	err := c.ImageManager.PullIfNotExist(cluster.Spec.Image)
	if err != nil {
		return err
	}
	return c.FileSystem.MountImage(cluster)
}

func (c Creation) RunConfig(cluster *v2.Cluster) error {
	return c.Config.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

func (c Creation) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0Ip())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return c.FileSystem.MountRootfs(cluster, hosts, true)
}

func (c Creation) Init(cluster *v2.Cluster) error {
	return c.Runtime.Init(cluster)
}

func (c Creation) Join(cluster *v2.Cluster) error {
	err := c.Runtime.JoinMasters(cluster.GetMasterIPList()[1:])
	if err != nil {
		return err
	}
	err = c.Runtime.JoinNodes(cluster.GetNodeIPList())
	if err != nil {
		return err
	}
	return nil
}

func (c Creation) RunGuest(cluster *v2.Cluster) error {
	return c.Guest.Apply(cluster)
}
func (c Creation) UnMountImage(cluster *v2.Cluster) error {
	return c.FileSystem.UnMountImage(cluster)
}

func (c Creation) initPlugin(cluster *v2.Cluster) error {
	c.Plugins = plugin.NewPlugins(cluster.Name)
	return c.Plugins.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

/*func (i Creation) RunPhasePlugin(cluster *v2.Cluster) error {
	if pluginPhases[0] == plugin.PhasePreInit {
		if err := i.Plugins.Load(); err != nil {
			return err
		}
	}
	err := i.Plugins.Run(cluster, pluginPhases[0])
	pluginPhases = pluginPhases[1:]
	return err
}*/

func NewCreateProcessor() (Interface, error) {
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

	return Creation{
		ImageManager: imgSvc,
		FileSystem:   fs,
		Guest:        gs,
	}, nil
}
