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
	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils"
)

type Upgrade struct {
	FileSystem    filesystem.Interface
	Runtime       runtime.Interface
	MastersToJoin []string
	NodesToJoin   []string
}

// DoApply do apply: do truly apply,input is desired cluster .
func (u Upgrade) Execute(cluster *v2.Cluster) error {
	runTime, err := runtime.NewDefaultRuntime(cluster, cluster.Annotations[common.ClusterfileName])
	if err != nil {
		return fmt.Errorf("failed to init runtime, %v", err)
	}
	u.Runtime = runTime
	err = u.MountRootfs(cluster)
	if err != nil {
		return err
	}
	err = u.Upgrade()
	if err != nil {
		return err
	}

	return nil
}

func (u Upgrade) MountRootfs(cluster *v2.Cluster) error {
	//some hosts already mounted when scaled cluster.
	currentHost := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	addedHost := append(u.MastersToJoin, u.NodesToJoin...)
	_, hosts := utils.GetDiffHosts(currentHost, addedHost)
	regConfig := runtime.GetRegistryConfig(common.DefaultTheClusterRootfsDir(cluster.Name), cluster.GetMaster0Ip())
	if utils.NotInIPList(regConfig.IP, hosts) {
		hosts = append(hosts, regConfig.IP)
	}
	return u.FileSystem.MountRootfs(cluster, hosts, false)
}

func (u Upgrade) Upgrade() error {
	return u.Runtime.Upgrade()
}

func NewUpgradeProcessor(fs filesystem.Interface, masterToJoin, nodeToJoin []string) (Interface, error) {
	// only do upgrade here. cancel scale action.
	return Upgrade{
		FileSystem:    fs,
		MastersToJoin: masterToJoin,
		NodesToJoin:   nodeToJoin,
	}, nil
}
