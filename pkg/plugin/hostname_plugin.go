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
	"net"
	"strings"

	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/ssh"
	"github.com/sirupsen/logrus"
)

type HostnamePlugin struct {
	// key: IP address of a node
	// value: hostname
	data map[string]string
}

func NewHostnamePlugin() Interface {
	return &HostnamePlugin{data: map[string]string{}}
}

func init() {
	Register(HostNamePlugin, NewHostnamePlugin())
}

func (h HostnamePlugin) Run(context Context, phase Phase) error {
	var err error
	if (phase != PhasePreInit && phase != PhasePreJoin) || context.Plugin.Spec.Type != HostNamePlugin {
		logrus.Debug("hostnamePlugin nodes is not PhasePreInit!")
		return nil
	}

	h.data, err = h.formatData(context.Plugin.Spec.Data)
	if err != nil {
		return fmt.Errorf("failed to format data from Plugin.Spec.Data: %v", err)
	}

	for ip, hostname := range h.data {
		if utilsnet.NotInIPList(net.ParseIP(ip), context.Host) {
			continue
		}
		sshClient, err := ssh.GetHostSSHClient(net.ParseIP(ip), context.Cluster)
		if err != nil {
			return err
		}
		err = h.changeNodeName(hostname, net.ParseIP(ip), sshClient)
		if err != nil {
			return fmt.Errorf("failed to update hostname of current cluster nodes(%s): %v", ip, err)
		}
	}
	return nil
}

func (h HostnamePlugin) formatData(data string) (map[string]string, error) {
	m := make(map[string]string)
	items := strings.Split(data, "\n")
	if len(items) == 0 {
		return nil, fmt.Errorf("hostname data(%s) cannot be empty", data)
	}
	for _, v := range items {
		tmps := strings.Split(v, " ")
		//skip no-compliance hostname data
		if len(tmps) != 2 {
			continue
		}
		ip := tmps[0]

		// validate the input IP to return fast
		if net.ParseIP(ip) == nil {
			return nil, fmt.Errorf("IP(%s) is an invalid IP", ip)
		}
		hostname := tmps[1]
		// TODO: add validation of hostname
		m[ip] = hostname
	}
	return m, nil
}

func (h HostnamePlugin) changeNodeName(hostname string, ip net.IP, SSH ssh.Interface) error {
	//cmd to change hostname temporarily
	tmpCMD := fmt.Sprintf("hostname %s", hostname)
	//cmd to change hostname permanently
	perCMD := fmt.Sprintf(`rm -f /etc/hostname && echo "%s" >> /etc/hostname`, hostname)
	if err := SSH.CmdAsync(ip, tmpCMD, perCMD); err != nil {
		return fmt.Errorf("failed to update hostname of node(%s): %v", ip, err)
	}
	logrus.Infof("succeed in updating hostname of node(%s) to %s", ip, hostname)
	return nil
}
