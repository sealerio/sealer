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
	"github.com/alibaba/sealer/pkg/filesystem"
	"github.com/alibaba/sealer/pkg/filesystem/cloudfilesystem"
	"github.com/alibaba/sealer/pkg/guest"
	v2 "github.com/alibaba/sealer/types/api/v2"
)

type InstallProcessor struct {
	fileSystem cloudfilesystem.Interface
	Guest      guest.Interface
}

// Execute :according to the different of desired cluster to install app on cluster.
func (i InstallProcessor) Execute(cluster *v2.Cluster) error {
	err := i.MountRootfs(cluster)
	if err != nil {
		return err
	}
	err = i.Install(cluster)
	if err != nil {
		return err
	}

	return nil
}

func (i InstallProcessor) MountRootfs(cluster *v2.Cluster) error {
	hosts := append(cluster.GetMasterIPList(), cluster.GetNodeIPList()...)
	//initFlag : no need to do init cmd like installing docker service and so on.
	return i.fileSystem.MountRootfs(cluster, hosts, false)
}

func (i InstallProcessor) Install(cluster *v2.Cluster) error {
	return i.Guest.Apply(cluster)
}

func NewInstallProcessor(rootfs string) (Interface, error) {
	gs, err := guest.NewGuestManager()
	if err != nil {
		return nil, err
	}

	fs, err := filesystem.NewFilesystem(rootfs)
	if err != nil {
		return nil, err
	}

	return InstallProcessor{
		fileSystem: fs,
		Guest:      gs,
	}, nil
}
