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

package apply

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/config"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/image"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/plugin"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// cloud builder using cloud provider to build a cluster image
type DefaultApplier struct {
	ClusterDesired  *v1.Cluster
	ClusterCurrent  *v1.Cluster
	ImageManager    image.Service
	FileSystem      filesystem.Interface
	Runtime         runtime.Interface
	Guest           guest.Interface
	Config          config.Interface
	Plugins         plugin.Plugins
	MastersToJoin   []string
	MastersToDelete []string
	NodesToJoin     []string
	NodesToDelete   []string
}

type ActionName string

const (
	PullIfNotExist            ActionName = "PullIfNotExist"
	MountRootfs               ActionName = "MountRootfs"
	UnMountRootfs             ActionName = "UnMountRootfs"
	MountImage                ActionName = "MountImage"
	Config                    ActionName = "Config"
	UnMountImage              ActionName = "UnMountImage"
	Init                      ActionName = "Init"
	Upgrade                   ActionName = "Upgrade"
	ApplyMasters              ActionName = "ApplyMasters"
	ApplyNodes                ActionName = "ApplyNodes"
	Guest                     ActionName = "Guest"
	Reset                     ActionName = "Reset"
	CleanFS                   ActionName = "CleanFS"
	PluginDump                ActionName = "PluginDump"
	PluginPhasePreInitRun     ActionName = "PluginPhasePreInitRun"
	PluginPhasePreInstallRun  ActionName = "PluginPhasePreInstallRun"
	PluginPhasePostInstallRun ActionName = "PluginPhasePostInstallRun"
)

var ActionFuncMap = map[ActionName]func(*DefaultApplier) error{
	PullIfNotExist: func(applier *DefaultApplier) error {
		imageName := applier.ClusterDesired.Spec.Image
		imgSvc, err := image.NewImageService()
		if err != nil {
			return err
		}
		return imgSvc.PullIfNotExist(imageName)
	},
	MountRootfs: func(applier *DefaultApplier) error {
		// TODO mount only mount desired hosts, some hosts already mounted when update cluster
		var hosts []string
		if applier.ClusterCurrent == nil {
			hosts = append(applier.ClusterDesired.Spec.Masters.IPList, applier.ClusterDesired.Spec.Nodes.IPList...)
			config := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(applier.ClusterDesired.Name), applier.ClusterDesired.Spec.Masters.IPList[0])
			if utils.NotInIPList(config.IP, applier.ClusterDesired.Spec.Masters.IPList) && utils.NotInIPList(config.IP, applier.ClusterDesired.Spec.Nodes.IPList) {
				hosts = append(hosts, config.IP)
			}
		} else {
			hosts = append(applier.MastersToJoin, applier.NodesToJoin...)
		}
		if err := applier.FileSystem.MountRootfs(applier.ClusterDesired, hosts); err != nil {
			return err
		}
		applier.Runtime = runtime.NewDefaultRuntime(applier.ClusterDesired)
		if applier.Runtime == nil {
			return fmt.Errorf("failed to init runtime")
		}
		return nil
	},
	UnMountRootfs: func(applier *DefaultApplier) error {
		return applier.FileSystem.UnMountRootfs(applier.ClusterDesired)
	},
	MountImage: func(applier *DefaultApplier) error {
		return applier.FileSystem.MountImage(applier.ClusterDesired)
	},
	Config: func(applier *DefaultApplier) error {
		return applier.Config.Dump(applier.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
	},
	UnMountImage: func(applier *DefaultApplier) error {
		return applier.FileSystem.UnMountImage(applier.ClusterDesired)
	},
	Init: func(applier *DefaultApplier) error {
		return applier.Runtime.Init(applier.ClusterDesired)
	},
	Upgrade: func(applier *DefaultApplier) error {
		return applier.Runtime.Upgrade(applier.ClusterDesired)
	},
	ApplyMasters: func(applier *DefaultApplier) error {
		return applyMasters(applier)
	},
	ApplyNodes: func(applier *DefaultApplier) error {
		return applyNodes(applier)
	},
	Guest: func(applier *DefaultApplier) error {
		return applier.Guest.Apply(applier.ClusterDesired)
	},
	Reset: func(applier *DefaultApplier) error {
		applier.Runtime = runtime.NewDefaultRuntime(applier.ClusterDesired)
		if applier.Runtime == nil {
			return fmt.Errorf("failed to init runtime")
		}
		return applier.Runtime.Reset(applier.ClusterDesired)
	},
	CleanFS: func(applier *DefaultApplier) error {
		return applier.FileSystem.Clean(applier.ClusterDesired)
	},
	PluginDump: func(applier *DefaultApplier) error {
		return applier.Plugins.Dump(applier.ClusterDesired.GetAnnotationsByKey(common.ClusterfileName))
	},
	PluginPhasePreInitRun: func(applier *DefaultApplier) error {
		return applier.Plugins.Run(applier.ClusterDesired, "PreInit")
	},
	PluginPhasePreInstallRun: func(applier *DefaultApplier) error {
		return applier.Plugins.Run(applier.ClusterDesired, "PreInstall")
	},
	PluginPhasePostInstallRun: func(applier *DefaultApplier) error {
		return applier.Plugins.Run(applier.ClusterDesired, "PostInstall")

	},
}

func applyMasters(applier *DefaultApplier) error {
	err := applier.Runtime.JoinMasters(applier.MastersToJoin)
	if err != nil {
		return err
	}
	err = applier.Runtime.DeleteMasters(applier.MastersToDelete)
	if err != nil {
		return err
	}
	return nil
}

func applyNodes(applier *DefaultApplier) error {
	err := applier.Runtime.JoinNodes(applier.NodesToJoin)
	if err != nil {
		return err
	}
	err = applier.Runtime.DeleteNodes(applier.NodesToDelete)
	if err != nil {
		return err
	}
	return nil
}

func (c *DefaultApplier) Apply() (err error) {
	if c.ClusterDesired.GetDeletionTimestamp().IsZero() {
		err = utils.SaveClusterfile(c.ClusterDesired)
		if err != nil {
			return err
		}

		currentCluster, err := GetCurrentCluster()
		if err != nil {
			return errors.Wrap(err, "get current cluster failed")
		}
		if currentCluster != nil {
			c.ClusterCurrent = c.ClusterDesired.DeepCopy()
			c.ClusterCurrent.Spec.Masters = currentCluster.Spec.Masters
			c.ClusterCurrent.Spec.Nodes = currentCluster.Spec.Nodes
		}
	}

	todoList, _ := c.diff()
	for _, action := range todoList {
		logger.Debug("sealer apply process %s", action)
		err := ActionFuncMap[action](c)
		if err != nil {
			if c.ClusterDesired.DeletionTimestamp != nil {
				logger.Warn(err)
			}else {
				return err
			}
		}
	}

	return nil
}

func (c *DefaultApplier) Delete() (err error) {
	t := metav1.Now()
	c.ClusterDesired.DeletionTimestamp = &t
	return c.Apply()
}

func (c *DefaultApplier) diff() (todoList []ActionName, err error) {
	if c.ClusterDesired.DeletionTimestamp != nil {
		c.MastersToDelete = c.ClusterDesired.Spec.Masters.IPList
		c.NodesToDelete = c.ClusterDesired.Spec.Nodes.IPList
		todoList = append(todoList, Reset)
		todoList = append(todoList, UnMountRootfs)
		todoList = append(todoList, UnMountImage)
		todoList = append(todoList, CleanFS)
		return todoList, nil
	}

	if c.ClusterCurrent == nil {
		todoList = append(todoList, PullIfNotExist)
		todoList = append(todoList, MountImage)
		todoList = append(todoList, PluginDump)
		todoList = append(todoList, Config)
		todoList = append(todoList, MountRootfs)
		todoList = append(todoList, PluginPhasePreInitRun)
		todoList = append(todoList, Init)
		todoList = append(todoList, PluginPhasePreInstallRun)
		c.MastersToJoin = c.ClusterDesired.Spec.Masters.IPList[1:]
		c.NodesToJoin = c.ClusterDesired.Spec.Nodes.IPList
		todoList = append(todoList, ApplyMasters)
		todoList = append(todoList, ApplyNodes)
		todoList = append(todoList, Guest)
		todoList = append(todoList, UnMountImage)
		todoList = append(todoList, PluginPhasePostInstallRun)
		return todoList, nil
	}

	todoList = append(todoList, PullIfNotExist)
	if c.ClusterDesired.Spec.Image != c.ClusterCurrent.Spec.Image {
		logger.Info("current image is : %s and desired image is : %s , so upgrade your cluster", c.ClusterCurrent.Spec.Image, c.ClusterDesired.Spec.Image)
		todoList = append(todoList, Upgrade)
	}
	c.MastersToJoin, c.MastersToDelete = utils.GetDiffHosts(c.ClusterCurrent.Spec.Masters, c.ClusterDesired.Spec.Masters)
	c.NodesToJoin, c.NodesToDelete = utils.GetDiffHosts(c.ClusterCurrent.Spec.Nodes, c.ClusterDesired.Spec.Nodes)
	todoList = append(todoList, MountImage)
	todoList = append(todoList, MountRootfs)
	if len(c.MastersToJoin) > 0 || len(c.MastersToDelete) > 0 {
		todoList = append(todoList, ApplyMasters)
	}
	if len(c.NodesToJoin) > 0 || len(c.NodesToDelete) > 0 {
		todoList = append(todoList, ApplyNodes)
	}
	todoList = append(todoList, Guest)
	todoList = append(todoList, UnMountImage)
	return todoList, nil
}

func NewDefaultApplier(cluster *v1.Cluster) (Interface, error) {
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
	return &DefaultApplier{
		ClusterDesired: cluster,
		ImageManager:   imgSvc,
		FileSystem:     fs,
		Guest:          gs,
		Config:         config.NewConfiguration(cluster.Name),
		Plugins:        plugin.NewPlugins(cluster.Name),
	}, nil
}
