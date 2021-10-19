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

package plugin

import (
	"fmt"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/utils"
	"github.com/alibaba/sealer/utils/ssh"
)

type Sheller struct{}

func NewShellPlugin() Interface {
	return &Sheller{}
}

func (s Sheller) Run(context Context, phase Phase) error {
	if string(phase) != context.Plugin.Spec.Action || context.Plugin.Spec.Type != ShellPlugin {
		return nil
	}
	//get cmdline content
	pluginCmd := context.Plugin.Spec.Data
	if phase != PhaseOriginally {
		pluginCmd = fmt.Sprintf(common.CdAndExecCmd, common.DefaultTheClusterRootfsDir(context.Cluster.Name), pluginCmd)
	}
	//get all host ip
	masterIP := context.Cluster.Spec.Masters.IPList
	nodeIP := context.Cluster.Spec.Nodes.IPList
	allHostIP := append(masterIP, nodeIP...)
	//get on

	if on := context.Plugin.Spec.On; on != "" {
		allHostIP = utils.DisassembleIPList(on)
	}

	sshClient, err := ssh.NewSSHClientWithCluster(context.Cluster)
	if err != nil {
		return err
	}
	for _, ip := range allHostIP {
		err := sshClient.SSH.CmdAsync(ip, pluginCmd)
		if err != nil {
			return fmt.Errorf("failed to run shell cmd,  %v", err)
		}
	}
	return nil
}
