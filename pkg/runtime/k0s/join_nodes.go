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
	"context"
	"fmt"
	"net"

	"github.com/sealerio/sealer/common"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) joinNodes(nodes []net.IP) error {
	if len(nodes) == 0 {
		return nil
	}
	if err := k.WaitSSHReady(6, nodes...); err != nil {
		return errors.Wrap(err, "join nodes wait for ssh ready time out")
	}
	/**To join a node, following these steps.
	STEP1: send private registry cert and add registry info into node
	STEP2: copy k0s join token
	STEP3: use k0s command to join node with worker role.
	STEP4: join node with token
	STEP5: start the k0sworker.service
	*/
	if err := k.sendRegistryCert(nodes); err != nil {
		return err
	}
	if err := k.CopyJoinToken(WorkerRole, nodes); err != nil {
		return err
	}
	addRegistryHostsAndLogin := k.addRegistryDomainToHosts()
	if k.RegConfig.Domain != common.DefaultRegistryDomain {
		addSeaHubHost := fmt.Sprintf("cat /etc/hosts | grep '%s' || echo '%s' >> /etc/hosts", k.RegConfig.IP.String()+" "+common.DefaultRegistryDomain, k.RegConfig.IP.String()+" "+common.DefaultRegistryDomain)
		addRegistryHostsAndLogin = fmt.Sprintf("%s && %s", addRegistryHostsAndLogin, addSeaHubHost)
	}
	if k.RegConfig.Username != "" && k.RegConfig.Password != "" {
		addRegistryHostsAndLogin = fmt.Sprintf("%s && %s", addRegistryHostsAndLogin, k.GenLoginCommand())
	}
	cmds := k.JoinCommand(WorkerRole)
	if cmds == nil {
		return fmt.Errorf("failed to get join node command")
	}

	eg, _ := errgroup.WithContext(context.Background())
	for _, node := range nodes {
		node := node
		eg.Go(func() error {
			logrus.Infof("start to join %s as worker", node)

			nodeCmds := append([]string{addRegistryHostsAndLogin}, cmds...)
			ssh, err := k.getHostSSHClient(node)
			if err != nil {
				return fmt.Errorf("failed to get node ssh client %s: %v", node, err)
			}
			if err := ssh.CmdAsync(node, nodeCmds...); err != nil {
				return fmt.Errorf("failed to join node %s: %v", node, err)
			}
			logrus.Infof("succeeded in joining %s as worker", node)
			return err
		})
	}
	return eg.Wait()
}
