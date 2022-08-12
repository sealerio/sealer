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

	"github.com/sealerio/sealer/utils/ssh"
	strUtils "github.com/sealerio/sealer/utils/strings"
	"github.com/sirupsen/logrus"
)

type HostnamePlugin struct {
	data map[string]string
}

func NewHostnamePlugin() Interface {
	return &HostnamePlugin{data: map[string]string{}}
}

func init() {
	Register(HostNamePlugin, NewHostnamePlugin())
}

func (h HostnamePlugin) Run(context Context, phase Phase) error {
	if (phase != PhasePreInit && phase != PhasePreJoin) || context.Plugin.Spec.Type != HostNamePlugin {
		logrus.Debug("hostnamePlugin nodes is not PhasePreInit!")
		return nil
	}
	h.data = h.formatData(context.Plugin.Spec.Data)
	for ip, hostname := range h.data {
		if strUtils.NotIn(ip, context.Host) {
			continue
		}
		sshClient, err := ssh.GetHostSSHClient(ip, context.Cluster, false)
		if err != nil {
			return err
		}
		err = h.changeNodeName(hostname, ip, sshClient)
		if err != nil {
			return fmt.Errorf("current cluster nodes hostname change failed, %v", err)
		}
	}
	return nil
}

func (h HostnamePlugin) formatData(data string) map[string]string {
	m := make(map[string]string)
	items := strings.Split(data, "\n")
	if len(items) == 0 {
		logrus.Debug("hostname data is empty!")
		return m
	}
	for _, v := range items {
		tmps := strings.Split(v, " ")
		//skip no-compliance hostname data
		if len(tmps) != 2 {
			continue
		}
		ip := tmps[0]
		hostname := tmps[1]
		m[ip] = hostname
	}
	return m
}

func (h HostnamePlugin) changeNodeName(hostname, ip string, SSH ssh.Interface) error {
	//cmd to change hostname temporarily
	tmpCMD := fmt.Sprintf("hostname %s", hostname)
	//cmd to change hostname permanently
	perCMD := fmt.Sprintf(`rm -f /etc/hostname && echo "%s" >> /etc/hostname`, hostname)
	if _, err := SSH.CmdAsync(ip, tmpCMD, perCMD); err != nil {
		return fmt.Errorf("failed to change the node %v hostname,%v", ip, err)
	}
	logrus.Infof("successfully changed node %s hostname to %s.", ip, hostname)
	return nil
}
