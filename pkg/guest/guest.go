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

	v1 "github.com/sealerio/sealer/pkg/define/image/v1"

	common2 "github.com/sealerio/sealer/pkg/define/options"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sealerio/sealer/pkg/imageengine"

	"github.com/sealerio/sealer/common"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"
)

type Interface interface {
	Apply(cluster *v2.Cluster) error
	Delete(cluster *v2.Cluster) error
}

type Default struct {
	ImageEngine imageengine.Interface
}

func NewGuestManager() (Interface, error) {
	ie, err := imageengine.NewImageEngine(common2.EngineGlobalConfigurations{})
	if err != nil {
		return nil, err
	}

	return &Default{ImageEngine: ie}, nil
}

func (d *Default) Apply(cluster *v2.Cluster) error {
	var (
		clusterRootfs = common.DefaultTheClusterRootfsDir(cluster.Name)
		ex            = shell.NewLex('\\')
		image         = cluster.Spec.Image
	)

	extension, err := d.ImageEngine.GetSealerImageExtension(&common2.GetImageAnnoOptions{ImageNameOrID: image})
	if err != nil {
		return fmt.Errorf("failed to get ClusterImage: %s", err)
	}
	//cmdArgs := d.getGuestCmdArg(cluster.Spec.CMDArgs, extension)
	cmd := d.getGuestCmd(cluster.Spec.CMD, extension)
	sshClient, err := ssh.NewStdoutSSHClient(cluster.GetMaster0IP(), cluster)
	if err != nil {
		return err
	}

	for _, value := range cmd {
		if value == "" {
			continue
		}
		cmdline, err := ex.ProcessWordWithMap(value, map[string]string{})
		if err != nil {
			return fmt.Errorf("failed to render build args: %v", err)
		}

		if err := sshClient.CmdAsync(cluster.GetMaster0IP(), fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

func (d *Default) getGuestCmd(CmdFromClusterFile []string, extension v1.ImageExtension) []string {
	if len(CmdFromClusterFile) > 0 {
		return CmdFromClusterFile
	}

	return extension.Launch.Cmds
}

//func (d *Default) getGuestCmdArg(clusterCmdsArgs []string, extension v1.ImageExtension) map[string]string {
//	var (
//		clusterArgs = clusterCmdsArgs
//		imageType   = extension.Type
//	)
//
//	if imageType == common.AppImage {
//		base = extension.Launch
//	}
//	base = extension.ArgSet
//	for k, v := range strings.ConvertToMap(clusterArgs) {
//		base[k] = v
//	}
//	return base
//}

func (d Default) Delete(cluster *v2.Cluster) error {
	panic("implement me")
}
