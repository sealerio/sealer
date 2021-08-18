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

	"github.com/alibaba/sealer/utils/ssh"
)

type Sheller struct {
}

func (s Sheller) Run(context Context, phase Phase) error {
	if string(phase) != context.Plugin.Spec.Action {
		return nil
	}
	//get cmdline content
	pluginData := context.Plugin.Spec.Data
	//get all host ip
	masterIP := context.Cluster.Spec.Masters.IPList
	nodeIP := context.Cluster.Spec.Nodes.IPList
	hostIP := append(masterIP, nodeIP...)

	SSH := ssh.NewSSHByCluster(context.Cluster)
	for _, ip := range hostIP {
		err := SSH.CmdAsync(ip, pluginData)
		if err != nil {
			return fmt.Errorf("failed to run shell cmd,  %v", err)
		}
	}
	return nil
}
