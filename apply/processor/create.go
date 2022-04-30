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

	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/filesystem/cloudimage"
	"github.com/sealerio/sealer/pkg/guest"
	"github.com/sealerio/sealer/pkg/image"
	"github.com/sealerio/sealer/pkg/plugin"
	"github.com/sealerio/sealer/pkg/runtime"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
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

func (c *CreateProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		c.MountImage,
		c.PreProcess,
		c.GetPhasePluginFunc(plugin.PhaseOriginally),
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

func (c *CreateProcessor) PreProcess(cluster *v2.Cluster) error {
	c.Config = config.NewConfiguration(cluster)
	c.initPlugin(cluster)
	return utils.SaveClusterInfoToFile(cluster, cluster.Name)
}

func (c *CreateProcessor) initPlugin(cluster *v2.Cluster) {
	c.Plugins = plugin.NewPlugins(cluster, c.ClusterFile.GetPlugins())
}

func (c *CreateProcessor) MountImage(cluster *v2.Cluster) error {
	platsMap, err := ssh.GetClusterPlatform(cluster)
	if err != nil {
		return err
	}
	plats := []*v1.Platform{platform.GetDefaultPlatform()}
	for _, v := range platsMap {
		plat := v
		plats = append(plats, &plat)
	}
	if err = c.ImageManager.PullIfNotExist(cluster.Spec.Image, plats); err != nil {
		return err
	}
	if err = c.cloudImageMounter.MountImage(cluster); err != nil {
		return err
	}
	runTime, err := runtime.NewDefaultRuntime(cluster, c.ClusterFile.GetKubeadmConfig())
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	c.Runtime = runTime
	return nil
}

func (c *CreateProcessor) RunConfig(cluster *v2.Cluster) error {
	return c.Config.Dump(c.ClusterFile.GetConfigs())
}

func (c *CreateProcessor) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	regConfig := runtime.GetRegistryConfig(platform.DefaultMountCloudImageDir(cluster.Name), cluster.GetMaster0IP())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}

	fs, err := filesystem.NewFilesystem(platform.DefaultMountCloudImageDir(cluster.Name))
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

func NewCreateProcessor(clusterFile clusterfile.Interface) (Processor, error) {
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
