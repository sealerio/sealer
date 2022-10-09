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

package application

import (
	"fmt"

	"github.com/sealerio/sealer/pkg/infradriver"

	common2 "github.com/sealerio/sealer/pkg/define/options"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sealerio/sealer/pkg/imageengine"

	"github.com/sealerio/sealer/common"
)

type Interface interface {
	Install(cmds []string) error
	Delete() error
}

type Default struct {
	infraDriver infradriver.InfraDriver
	imageEngine imageengine.Interface
}

func NewAppInstaller(infra infradriver.InfraDriver) (Interface, error) {
	ie, err := imageengine.NewImageEngine(common2.EngineGlobalConfigurations{})
	if err != nil {
		return nil, err
	}

	return &Default{imageEngine: ie, infraDriver: infra}, nil
}

func (d *Default) Install(cmds []string) error {
	var (
		clusterRootfs = d.infraDriver.GetClusterRootfsPath()
		ex            = shell.NewLex('\\')
		image         = d.infraDriver.GetClusterImageName()
		cmd           []string
		master0       = d.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	)

	extension, err := d.imageEngine.GetSealerImageExtension(&common2.GetImageAnnoOptions{ImageNameOrID: image})
	if err != nil {
		return fmt.Errorf("failed to get ClusterImage: %s", err)
	}

	if len(cmds) > 0 {
		cmd = cmds
	} else {
		cmd = extension.Launch.Cmds
	}

	for _, value := range cmd {
		if value == "" {
			continue
		}
		cmdline, err := ex.ProcessWordWithMap(value, map[string]string{})
		if err != nil {
			return fmt.Errorf("failed to render build args: %v", err)
		}

		if err = d.infraDriver.CmdAsync(master0, fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, cmdline)); err != nil {
			return err
		}
	}

	return nil
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

func (d Default) Delete() error {
	panic("implement me")
}
