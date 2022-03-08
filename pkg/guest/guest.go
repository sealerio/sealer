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
	"strings"

	v1 "github.com/alibaba/sealer/types/api/v1"

	"github.com/alibaba/sealer/utils"

	"github.com/moby/buildkit/frontend/dockerfile/shell"

	"github.com/alibaba/sealer/pkg/runtime"
	v2 "github.com/alibaba/sealer/types/api/v2"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/image/store"
	"github.com/alibaba/sealer/utils/ssh"
)

type Interface interface {
	Apply(cluster *v2.Cluster) error
	Delete(cluster *v2.Cluster) error
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

func (d *Default) Apply(cluster *v2.Cluster) error {
	var (
		clusterRootfs = common.DefaultTheClusterRootfsDir(cluster.Name)
		ex            = shell.NewLex('\\')
	)

	image, err := d.imageStore.GetByName(cluster.Spec.Image)
	if err != nil {
		return fmt.Errorf("get cluster image failed, %s", err)
	}
	cmdArgs := d.getGuestCmdArg(cluster, image)
	cmd := d.getGuestCmd(cluster, image)

	sshClient, err := ssh.NewStdoutSSHClient(runtime.GetMaster0Ip(cluster), cluster)
	if err != nil {
		return err
	}

	for _, value := range cmd {
		cmdline, err := ex.ProcessWordWithMap(value, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to render build args: %v", err)
		}

		if err := sshClient.CmdAsync(runtime.GetMaster0Ip(cluster), fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

func (d *Default) getGuestCmd(cluster *v2.Cluster, image *v1.Image) []string {
	if len(cluster.Spec.CMD) != 0 {
		return cluster.Spec.CMD
	}

	var cmd []string
	for i := range image.Spec.Layers {
		if image.Spec.Layers[i].Type != common.CMDCOMMAND {
			continue
		}
		cmdValue := image.Spec.Layers[i].Value
		cmd = append(cmd, strings.Split(cmdValue, ",")...)
	}
	return cmd
}

func (d *Default) getGuestCmdArg(cluster *v2.Cluster, image *v1.Image) map[string]string {
	var args []string
	if image.Spec.ImageConfig.Args != nil {
		args = append(args, utils.ConvertMapToEnvList(image.Spec.ImageConfig.Args)...)
	}
	if len(cluster.Spec.CMDArgs) != 0 {
		args = append(args, cluster.Spec.CMDArgs...)
	}
	return utils.ConvertEnvListToMap(args)
}

func (d Default) Delete(cluster *v2.Cluster) error {
	panic("implement me")
}
