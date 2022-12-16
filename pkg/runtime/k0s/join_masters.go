// Copyright © 2022 Alibaba Group Holding Ltd.
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

package k0s

import (
	"fmt"
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/utils/ssh"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func (k *Runtime) joinMasters(masters []net.IP) error {
	if len(masters) == 0 {
		return nil
	}
	if err := k.WaitSSHReady(6, masters...); err != nil {
		return errors.Wrap(err, "join masters wait for ssh ready time out")
	}
	/**To join a node, following these steps.
	STEP1: send private registry cert and add registry info into node
	STEP2: copy k0s join token
	STEP3: use k0s command to join node with master role.
	STEP4: k0s create default config
	STEP5: modify the private image repository field and so on in k0s config
	STEP6: join node with token
	STEP7: start the k0scontroller.service
	*/
	if err := k.sendRegistryCert(masters); err != nil {
		return err
	}
	if err := k.CopyJoinToken(ControllerRole, masters); err != nil {
		return err
	}
	cmds := k.JoinCommand(ControllerRole)
	if cmds == nil {
		return fmt.Errorf("failed to get join master command")
	}

	for _, master := range masters {
		logrus.Infof("start to join %s as master", master)

		masterCmds := k.JoinMasterCommands(cmds)
		client, err := k.getHostSSHClient(master)
		if err != nil {
			return err
		}

		if client.(*ssh.SSH).User != common.ROOT {
			masterCmds = append(masterCmds, "rm -rf ${HOME}/.kube/config && mkdir -p ${HOME}/.kube && cp /var/lib/k0s/pki/admin.conf ${HOME}/.kube/config && chown $(id -u):$(id -g) ${HOME}/.kube/config")
		}

		if err := client.CmdAsync(master, masterCmds...); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", cmds, master, err)
		}

		logrus.Infof("succeeded in joining %s as master", master)
	}
	return nil
}

func (k *Runtime) JoinMasterCommands(cmds []string) []string {
	cmdAddRegistryHosts := k.addRegistryDomainToHosts()
	if k.RegConfig.Domain != common.DefaultRegistryDomain {
		cmdAddSeaHubHosts := fmt.Sprintf("cat /etc/hosts | grep '%s' || echo '%s' >> /etc/hosts", k.RegConfig.IP.String()+" "+common.DefaultRegistryDomain, k.RegConfig.IP.String()+" "+common.DefaultRegistryDomain)
		cmdAddRegistryHosts = fmt.Sprintf("%s && %s", cmdAddRegistryHosts, cmdAddSeaHubHosts)
	}
	joinCommands := []string{cmdAddRegistryHosts}
	if k.RegConfig.Username != "" && k.RegConfig.Password != "" {
		joinCommands = append(joinCommands, k.GenLoginCommand())
	}

	return append(joinCommands, cmds...)
}
