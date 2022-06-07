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
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/guest"
	"github.com/sealerio/sealer/pkg/plugin"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
)

type InstallProcessor struct {
	clusterFile clusterfile.Interface
	Guest       guest.Interface
	Config      config.Interface
	Plugins     plugin.Plugins
}

func (i *InstallProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	todoList = append(todoList,
		i.Process,
		i.RunConfig,
		i.MountRootfs,
		i.GetPhasePluginFunc(plugin.PhasePreGuest),
		i.Install,
		i.GetPhasePluginFunc(plugin.PhasePostInstall),
	)
	return todoList, nil
}

func (i *InstallProcessor) Process(cluster *v2.Cluster) error {
	i.Config = config.NewConfiguration(cluster)
	i.Plugins = plugin.NewPlugins(cluster, i.clusterFile.GetPlugins())
	return nil
}

func (i *InstallProcessor) RunConfig(cluster *v2.Cluster) error {
	return i.Config.Dump(i.clusterFile.GetConfigs())
}

func (i *InstallProcessor) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	//initFlag : no need to do init cmd like installing docker service and so on.
	fs, err := filesystem.NewFilesystem(platform.DefaultMountClusterImageDir(cluster.Name))
	if err != nil {
		return err
	}
	return fs.MountRootfs(cluster, hosts, false)
}

func (i *InstallProcessor) Install(cluster *v2.Cluster) error {
	return i.Guest.Apply(cluster)
}

func (i *InstallProcessor) GetPhasePluginFunc(phase plugin.Phase) func(cluster *v2.Cluster) error {
	return func(cluster *v2.Cluster) error {
		if phase == plugin.PhasePreGuest {
			if err := i.Plugins.Load(); err != nil {
				return err
			}
		}
		return i.Plugins.Run(cluster.GetAllIPList(), phase)
	}
}

func NewInstallProcessor(clusterFile clusterfile.Interface) (Processor, error) {
	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}

	return &InstallProcessor{
		clusterFile: clusterFile,
		Guest:       gs,
	}, nil
}
