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
	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/filesystem"
	"github.com/sealerio/sealer/pkg/filesystem/cloudfilesystem"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/net"
)

type UpgradeProcessor struct {
	fileSystem cloudfilesystem.Interface
	Runtime    runtime.Interface
}

// Execute :according to the different of desired cluster to upgrade cluster.
func (u UpgradeProcessor) Execute(cluster *v2.Cluster) error {
	err := u.MountRootfs(cluster)
	if err != nil {
		return err
	}
	err = u.Upgrade()
	if err != nil {
		return err
	}

	return nil
}

func (u UpgradeProcessor) MountRootfs(cluster *v2.Cluster) error {
	//some hosts already mounted when scaled cluster.
	hosts := cluster.GetAllIPList()
	regConfig := registry.GetConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0IP())
	if net.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return u.fileSystem.MountRootfs(cluster, hosts, false)
}

func (u UpgradeProcessor) Upgrade() error {
	return u.Runtime.Upgrade()
}

func NewUpgradeProcessor(rootfs string, rt runtime.Interface) (Interface, error) {
	// only do upgrade here. cancel scale action.
	fs, err := filesystem.NewFilesystem(rootfs)
	if err != nil {
		return nil, err
	}

	return UpgradeProcessor{
		fileSystem: fs,
		Runtime:    rt,
	}, nil
}
