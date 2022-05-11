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
	"strings"

	"github.com/sealerio/sealer/utils/slice"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/utils/ssh"
)

type Sheller struct{}

func NewShellPlugin() Interface {
	return &Sheller{}
}

func init() {
	Register(ShellPlugin, NewShellPlugin())
}

func (s Sheller) Run(context Context, phase Phase) (err error) {
	pluginPhases := strings.Split(context.Plugin.Spec.Action, SplitSymbol)
	if slice.NotIn(string(phase), pluginPhases) || context.Plugin.Spec.Type != ShellPlugin {
		return nil
	}
	//get cmdline content
	pluginCmd := context.Plugin.Spec.Data
	if phase != PhaseOriginally {
		pluginCmd = fmt.Sprintf(common.CdAndExecCmd, common.DefaultTheClusterRootfsDir(context.Cluster.Name), pluginCmd)
	}
	//get all host ip
	allHostIP := append(context.Cluster.GetMasterIPList(), context.Cluster.GetNodeIPList()...)
	if on := context.Plugin.Spec.On; on != "" {
		allHostIP, err = GetIpsByOnField(on, context, phase)
		if err != nil {
			if phase == PhasePreClean {
				logger.Error("failed to get ips when %s phase: %v", phase, err)
				return nil
			}
			return err
		}
	}
	var runPluginIPList []string
	for _, ip := range allHostIP {
		//skip non-cluster nodes
		if slice.NotIn(ip, context.Host) {
			continue
		}
		envProcessor := env.NewEnvProcessor(context.Cluster)
		sshClient, err := ssh.NewStdoutSSHClient(ip, context.Cluster)
		if err != nil {
			return err
		}
		err = sshClient.CmdAsync(ip, envProcessor.WrapperShell(ip, pluginCmd))
		if err != nil {
			return fmt.Errorf("failed to run shell cmd,  %v", err)
		}
		runPluginIPList = append(runPluginIPList, ip)
	}
	logger.Info("%s phase shell plugin '%s' executing nodes: %s ", phase, context.Plugin.Name, runPluginIPList)
	return nil
}
