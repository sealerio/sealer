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

package guest

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/image/store"
	v1 "github.com/alibaba/sealer/types/api/v1"
	ssh2 "github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	Apply(cluster *v1.Cluster) error
	Delete(cluster *v1.Cluster) error
}

type Default struct {
	imageStore store.ImageStore
}

func NewGuestManager() (Interface, error) {
	is, err := store.NewDefaultImageStore()
	if err != nil {
		return nil, err
	}

	return &Default{imageStore: is}, nil
}

func (d *Default) Apply(cluster *v1.Cluster) error {
	ssh := ssh2.NewSSHByCluster(cluster)
	image, err := d.imageStore.GetByName(cluster.Spec.Image)
	if err != nil {
		return fmt.Errorf("get cluster image failed, %s", err)
	}
	masters := cluster.Spec.Masters.IPList
	if len(masters) == 0 {
		return fmt.Errorf("failed to found master")
	}
	clusterRootfs := common.DefaultTheClusterRootfsDir(cluster.Name)
	for i := range image.Spec.Layers {
		if image.Spec.Layers[i].Type != common.CMDCOMMAND {
			continue
		}
		if err := ssh.CmdAsync(masters[0], fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, image.Spec.Layers[i].Value)); err != nil {
			return err
		}
	}
	return nil
}

func (d Default) Delete(cluster *v1.Cluster) error {
	panic("implement me")
}
