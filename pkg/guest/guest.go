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

	"github.com/alibaba/sealer/pkg/runtime"
	"github.com/alibaba/sealer/utils/platform"

	v1 "github.com/alibaba/sealer/types/api/v1"

	"github.com/alibaba/sealer/utils"

	"github.com/moby/buildkit/frontend/dockerfile/shell"

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

	image, err := d.imageStore.GetByName(cluster.Spec.Image, platform.GetDefaultPlatform())
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
		if value == "" {
			continue
		}
		cmdline, err := ex.ProcessWordWithMap(value, cmdArgs)
		if err != nil {
			return fmt.Errorf("failed to render build args: %v", err)
		}

		if err := sshClient.CmdAsync(cluster.GetMaster0IP(), fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

func (d *Default) getGuestCmd(cluster *v2.Cluster, image *v1.Image) []string {
	var (
		cmd        = image.Spec.ImageConfig.Cmd.Parent
		clusterCmd = cluster.Spec.CMD
		imageType  = image.Spec.ImageConfig.ImageType
	)

	// application image: if cluster cmd not nil, use cluster cmd directly
	if imageType == common.AppImage {
		if len(clusterCmd) != 0 {
			return clusterCmd
		}
		return image.Spec.ImageConfig.Cmd.Current
	}

	// normal image: if cluster cmd not nil, use cluster cmd as current cmd
	if len(clusterCmd) != 0 {
		return utils.MergeSlice(cmd, clusterCmd)
	}
	return utils.MergeSlice(cmd, image.Spec.ImageConfig.Cmd.Current)
}

func (d *Default) getGuestCmdArg(cluster *v2.Cluster, image *v1.Image) map[string]string {
	var (
		base        map[string]string
		clusterArgs = cluster.Spec.CMDArgs
		imageType   = image.Spec.ImageConfig.ImageType
	)

	if imageType == common.AppImage {
		base = image.Spec.ImageConfig.Args.Current
	} else {
		base = utils.MergeMap(image.Spec.ImageConfig.Args.Parent, image.Spec.ImageConfig.Args.Current)
	}

	for k, v := range utils.ConvertEnvListToMap(clusterArgs) {
		base[k] = v
	}
	return base
}

func (d Default) Delete(cluster *v2.Cluster) error {
	panic("implement me")
}
