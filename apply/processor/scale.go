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
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/filesystem/cloudfilesystem"
	"github.com/sealerio/sealer/pkg/plugin"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v2 "github.com/sealerio/sealer/types/api/v2"
	platform "github.com/sealerio/sealer/utils/platform"
)

type ScaleProcessor struct {
	fileSystem      cloudfilesystem.Interface
	ClusterFile     clusterfile.Interface
	Runtime         runtime.Interface
	KubeadmConfig   *kubeadm.KubeadmConfig
	Config          config.Interface
	Plugins         plugin.Plugins
	MastersToJoin   []net.IP
	MastersToDelete []net.IP
	NodesToJoin     []net.IP
	NodesToDelete   []net.IP
	IsScaleUp       bool
}

func (s *ScaleProcessor) GetPipeLine() ([]func(cluster *v2.Cluster) error, error) {
	var todoList []func(cluster *v2.Cluster) error
	if s.IsScaleUp {
		todoList = append(todoList,
			s.PreProcess,
			s.GetPhasePluginFunc(plugin.PhaseOriginally),
			s.RunConfig,
			s.MountRootfs,
			s.GetPhasePluginFunc(plugin.PhasePreJoin),
			s.Join,
			s.GetPhasePluginFunc(plugin.PhasePreGuest), //taint plugin, label plugin, or clusterCheck plugin
			s.GetPhasePluginFunc(plugin.PhasePostJoin),
		)
		return todoList, nil
	}

	todoList = append(todoList,
		s.PreProcess,
		s.GetPhasePluginFunc(plugin.PhasePreClean),
		s.Delete,
		s.GetPhasePluginFunc(plugin.PhasePostClean),
		s.UnMountRootfs,
	)
	return todoList, nil
}

func (s *ScaleProcessor) PreProcess(cluster *v2.Cluster) error {
	s.Config = config.NewConfiguration(platform.DefaultMountClusterImageDir(cluster.Name))
	if s.IsScaleUp {
		if err := clusterfile.SaveToDisk(cluster, cluster.Name); err != nil {
			return err
		}
	}
	return s.initPlugin(cluster)
}

func (s *ScaleProcessor) initPlugin(cluster *v2.Cluster) error {
	s.Plugins = plugin.NewPlugins(cluster, s.ClusterFile.GetPlugins())
	return s.Plugins.Load()
}

func (s *ScaleProcessor) GetPhasePluginFunc(phase plugin.Phase) func(cluster *v2.Cluster) error {
	return func(cluster *v2.Cluster) error {
		if s.IsScaleUp {
			return s.Plugins.Run(append(s.MastersToJoin, s.NodesToJoin...), phase)
		}
		return s.Plugins.Run(append(s.MastersToDelete, s.NodesToDelete...), phase)
	}
}

func (s *ScaleProcessor) RunConfig(cluster *v2.Cluster) error {
	return s.Config.Dump(s.ClusterFile.GetConfigs())
}

func (s *ScaleProcessor) MountRootfs(cluster *v2.Cluster) error {
	return s.fileSystem.MountRootfs(cluster, append(s.MastersToJoin, s.NodesToJoin...), true)
}

func (s *ScaleProcessor) UnMountRootfs(cluster *v2.Cluster) error {
	return s.fileSystem.UnMountRootfs(cluster, append(s.MastersToDelete, s.NodesToDelete...))
}

func (s *ScaleProcessor) Join(cluster *v2.Cluster) error {
	if err := s.Runtime.JoinMasters(s.MastersToJoin); err != nil {
		return err
	}
	return s.Runtime.JoinNodes(s.NodesToJoin)
}

func (s *ScaleProcessor) Delete(cluster *v2.Cluster) error {
	err := s.Runtime.DeleteMasters(s.MastersToDelete)
	if err != nil {
		return err
	}
	return s.Runtime.DeleteNodes(s.NodesToDelete)
}

func NewScaleProcessor(kubeadmConfig *kubeadm.KubeadmConfig, clusterFile clusterfile.Interface, masterToJoin, masterToDelete, nodeToJoin, nodeToDelete []net.IP) (Processor, error) {
	cluster := clusterFile.GetCluster()
	fs, err := filesystem.NewFilesystem(common.DefaultTheClusterRootfsDir(cluster.Name))
	if err != nil {
		return nil, err
	}

	var up bool
	// only scale up or scale down at a time
	if len(masterToJoin) > 0 || len(nodeToJoin) > 0 {
		up = true
	}

	runTime, err := ChooseRuntime(platform.DefaultMountClusterImageDir(cluster.Name), &cluster, clusterFile.GetKubeadmConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to init default runtime: %v", err)
	}

	return &ScaleProcessor{
		Runtime:         runTime,
		MastersToDelete: masterToDelete,
		MastersToJoin:   masterToJoin,
		NodesToDelete:   nodeToDelete,
		NodesToJoin:     nodeToJoin,
		KubeadmConfig:   kubeadmConfig,
		ClusterFile:     clusterFile,
		IsScaleUp:       up,
		fileSystem:      fs,
	}, nil
}
