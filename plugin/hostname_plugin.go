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

	"github.com/alibaba/sealer/client"
	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils/ssh"

	v1 "k8s.io/api/core/v1"
)

/*
labels plugin in Clusterfile:
---
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: HOSTNAME
spec:
  data: |
     192.168.0.2 master-0
     192.168.0.3 master-1
     192.168.0.4 master-2
     192.168.0.5 node-0
     192.168.0.6 node-1
     192.168.0.7 node-2
---
HostnamePlugin.data
key = ip
value = target hostname
*/
type HostnamePlugin struct {
	data map[string]string
}

func NewHostNamePlugin() Interface {
	return &HostnamePlugin{
		data: map[string]string{},
	}
}

func (h HostnamePlugin) Run(context Context, phase Phase) error {
	if phase != PhasePreInstall {
		logger.Debug("label nodes is not PreInit!")
		return nil
	}
	h.data = h.formatData(context.Plugin.Spec.Data)
	c, err := client.NewClientSet()
	if err != nil {
		return fmt.Errorf("current cluster not found, %v", err)
	}
	nodeList, err := client.ListNodes(c)
	if err != nil {
		return fmt.Errorf("current cluster nodes not found, %v", err)
	}
	for _, v := range nodeList.Items {
		internalIP := h.getAddress(v.Status.Addresses)
		hostname, ok := h.data[internalIP]
		SSH := ssh.NewSSHByCluster(context.Cluster)
		if ok {
			if err := h.changeNodeName(hostname, internalIP, SSH); err != nil {
				return fmt.Errorf("change the hostname of the current cluster nodes failed, %v", err)
			}
		}
	}
	return err
}

func (h HostnamePlugin) formatData(data string) map[string]string {
	m := make(map[string]string)
	items := strings.Split(data, "\n")
	if len(items) == 0 {
		logger.Debug("label data is empty!")
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

func (h HostnamePlugin) getAddress(addresses []v1.NodeAddress) string {
	for _, v := range addresses {
		if strings.EqualFold(string(v.Type), "InternalIP") {
			return v.Address
		}
	}
	return ""
}

func (h HostnamePlugin) changeNodeName(hostName, ip string, SSH ssh.Interface) error {
	//cmd to change hostName temporarily,the change will
	tmpCMD := fmt.Sprintf("hostname %s", hostName)
	perCMD := fmt.Sprintf(`rm -f /etc/hostname && echo "%s" >> /etc/hostname`, hostName)
	if _, err := SSH.Cmd(ip, tmpCMD); err != nil {
		return err
	}
	if _, err := SSH.Cmd(ip, perCMD); err != nil {
		return err
	}
	return nil
}
