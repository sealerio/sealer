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
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/ssh"
	strUtils "github.com/sealerio/sealer/utils/strings"
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
	if strUtils.NotIn(string(phase), pluginPhases) || context.Plugin.Spec.Type != ShellPlugin {
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
				logrus.Errorf("failed to get ips when %s phase: %v", phase, err)
				return nil
			}
			return err
		}
	}
	logrus.Infof("Start to run %s-shell-plugin '%s', this may take a few minutes, please be patient...", phase, context.Plugin.Name)

	var runPluginIPList []string
	var wg sync.WaitGroup
	hostErrRecorder := utils.NewHostErrRecorder()

	envProcessor := env.NewEnvProcessor(context.Cluster)
	for _, ip := range allHostIP {
		//skip non-cluster nodes
		if strUtils.NotIn(ip, context.Host) {
			continue
		}
		runPluginIPList = append(runPluginIPList, ip)

		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			sshClient, err := ssh.GetHostSSHClient(node, context.Cluster, true)
			if err != nil {
				hostErrRecorder.AppendErr(node, err)

				return
			}
			_, err = sshClient.CmdAsync(node, envProcessor.WrapperShell(node, pluginCmd))
			if err != nil {
				hostErrRecorder.AppendErr(node, err)

				return
			}
		}(ip)
	}
	wg.Wait()

	if err := hostErrRecorder.Result(); err != nil {
		return err
	}

	logrus.Infof("Succeeded in running %s-shell-plugin '%s' on nodes: %v", phase, context.Plugin.Name, runPluginIPList)

	return nil
}
