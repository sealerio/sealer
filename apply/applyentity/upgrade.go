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

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/filesystem"
	"github.com/alibaba/sealer/guest"
	"github.com/alibaba/sealer/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type UpgradeApply struct {
	FileSystem filesystem.Interface
	Runtime    runtime.Interface
	Guest      guest.Interface
}

// DoApply do apply: do truly apply,input is desired cluster .
func (u UpgradeApply) DoApply(cluster *v1.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster)
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	u.Runtime = runTime
	err = u.MountRootfs(cluster)
	if err != nil {
		return err
	}
	err = u.Upgrade(cluster)
	if err != nil {
		return err
	}
	err = u.RunGuest(cluster)
	if err != nil {
		return err
	}

	return nil
}

func (u UpgradeApply) MountRootfs(cluster *v1.Cluster) error {
	// TODO mount only mount desired hosts, some hosts already mounted when update cluster
	var hosts []string
	hosts = append(cluster.Spec.Masters.IPList, cluster.Spec.Nodes.IPList...)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.Spec.Masters.IPList[0])
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return u.FileSystem.MountRootfs(cluster, hosts, false)
}

func (u UpgradeApply) Upgrade(cluster *v1.Cluster) error {
	return u.Runtime.Upgrade(cluster)
}

func (u UpgradeApply) RunGuest(cluster *v1.Cluster) error {
	return u.Guest.Apply(cluster)
}

func NewUpgradeApply(fs filesystem.Interface) (Interface, error) {
	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}
	// only do upgrade here. cancel scale action.
	return UpgradeApply{
		FileSystem: fs,
		Guest:      gs,
	}, nil
}
