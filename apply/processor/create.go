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
	"github.com/alibaba/sealer/pkg/plugin"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
)

type CreateProcessor struct {
	ImageManager image.Service
	FileSystem   filesystem.Interface
	Runtime      runtime.Interface
	Guest        guest.Interface
	Config       config.Interface
	Plugins      plugin.Plugins
}

func (c *CreateProcessor) Execute(cluster *v2.Cluster) error {
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
func (c *CreateProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		c.GetPhasePluginFunc(plugin.PhaseOriginally),
		c.MountImage,
		c.RunConfig,
		c.MountRootfs,
		c.GetPhasePluginFunc(plugin.PhasePreInit),
		c.Init,
		c.Join,
		c.GetPhasePluginFunc(plugin.PhasePreGuest),
		c.RunGuest,
		c.UnMountImage,
		c.GetPhasePluginFunc(plugin.PhasePostInstall),
	)
	return todoList, nil
}

func (c *CreateProcessor) MountImage(cluster *v2.Cluster) error {
	err := c.ImageManager.PullIfNotExist(cluster.Spec.Image)
	if err != nil {
		return err
	}
	return c.FileSystem.MountImage(cluster)
}

func (c *CreateProcessor) RunConfig(cluster *v2.Cluster) error {
	return c.Config.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

func (c *CreateProcessor) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0Ip())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return c.FileSystem.MountRootfs(cluster, hosts, true)
}

func (c *CreateProcessor) Init(cluster *v2.Cluster) error {
	return c.Runtime.Init(cluster)
}

func (c *CreateProcessor) Join(cluster *v2.Cluster) error {
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

func (c *CreateProcessor) RunGuest(cluster *v2.Cluster) error {
	return c.Guest.Apply(cluster)
}
func (c *CreateProcessor) UnMountImage(cluster *v2.Cluster) error {
	return c.FileSystem.UnMountImage(cluster)
}

func (c *CreateProcessor) initPlugin(cluster *v2.Cluster) error {
	c.Plugins = plugin.NewPlugins(cluster.Name)
	return c.Plugins.Dump(cluster.GetAnnotationsByKey(common.ClusterfileName))
}

func (c *CreateProcessor) GetPhasePluginFunc(phase plugin.Phase) func(cluster *v2.Cluster) error {
	return func(cluster *v2.Cluster) error {
		if phase == plugin.PhasePreInit {
			if err := c.Plugins.Load(); err != nil {
				return err
			}
		}
		return c.Plugins.Run(cluster, phase)
	}
}

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

	return &CreateProcessor{
		ImageManager: imgSvc,
		FileSystem:   fs,
		Guest:        gs,
	}, nil
}
