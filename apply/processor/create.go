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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/clusterfile"
	"github.com/alibaba/sealer/pkg/config"
	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/filesystem/cloudimage"
	"github.com/alibaba/sealer/pkg/guest"
	"github.com/alibaba/sealer/pkg/image"
	"github.com/alibaba/sealer/pkg/plugin"
	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

type CreateProcessor struct {
	ClusterFile       clusterfile.Interface
	ImageManager      image.Service
	cloudImageMounter cloudimage.Interface
	Runtime           runtime.Interface
	Guest             guest.Interface
	Config            config.Interface
	Plugins           plugin.Plugins
}

func (c *CreateProcessor) Execute(cluster *v2.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster, c.ClusterFile.GetKubeadmConfig())
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	c.Runtime = runTime
	c.Config = config.NewConfiguration(cluster)
	if err := c.initPlugin(cluster); err != nil {
		return err
	}
	err = utils.SaveClusterInfoToFile(cluster, cluster.Name)
	if err != nil {
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

	return c.cloudImageMounter.MountImage(cluster)
}

func (c *CreateProcessor) RunConfig(cluster *v2.Cluster) error {
	return c.Config.Dump(c.ClusterFile.GetConfigs())
}

func (c *CreateProcessor) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0IP())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}

	fs, err := filesystem.NewFilesystem(common.DefaultMountCloudImageDir(cluster.Name))
	if err != nil {
		return err
	}

	return fs.MountRootfs(cluster, hosts, true)
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
	return utils.SaveClusterInfoToFile(cluster, cluster.Name)
}

func (c *CreateProcessor) RunGuest(cluster *v2.Cluster) error {
	return c.Guest.Apply(cluster)
}
func (c *CreateProcessor) UnMountImage(cluster *v2.Cluster) error {
	return c.cloudImageMounter.UnMountImage(cluster)
}

func (c *CreateProcessor) initPlugin(cluster *v2.Cluster) error {
	c.Plugins = plugin.NewPlugins(cluster)
	return c.Plugins.Dump(c.ClusterFile.GetPlugins())
}

func (c *CreateProcessor) GetPhasePluginFunc(phase plugin.Phase) func(cluster *v2.Cluster) error {
	return func(cluster *v2.Cluster) error {
		if phase == plugin.PhasePreInit {
			if err := c.Plugins.Load(); err != nil {
				return err
			}
		}
		return c.Plugins.Run(cluster.GetAllIPList(), phase)
	}
}

func NewCreateProcessor(clusterFile clusterfile.Interface) (Interface, error) {
	imgSvc, err := image.NewImageService()
	if err != nil {
		return nil, err
	}

	mounter, err := filesystem.NewCloudImageMounter()
	if err != nil {
		return nil, err
	}

	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}

	return &CreateProcessor{
		ClusterFile:       clusterFile,
		ImageManager:      imgSvc,
		cloudImageMounter: mounter,
		Guest:             gs,
	}, nil
}
