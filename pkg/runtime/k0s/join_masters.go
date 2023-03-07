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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const TimeForWaitingK0sStart = 10

func (k *Runtime) joinMasters(masters []net.IP, registryInfo string) error {
	if len(masters) == 0 {
		return nil
	}

	if err := k.initKube(masters); err != nil {
		return err
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
	if err := k.CopyJoinToken(ControllerRole, masters); err != nil {
		return err
	}
	cmds := k.JoinCommand(ControllerRole, registryInfo)
	if cmds == nil {
		return fmt.Errorf("failed to get join master command")
	}

	for _, master := range masters {
		logrus.Infof("Start to join %s as master", master)
		if err := k.infra.CmdAsync(master, nil, cmds...); err != nil {
			return fmt.Errorf("failed to exec command(%s) on master(%s): %v", cmds, master, err)
		}
		if err := k.fetchKubeconfig(master); err != nil {
			return fmt.Errorf("failed to fetch admin.conf on master(%s): %v", master, err)
		}
		logrus.Infof("Succeeded in joining %s as master", master)
	}
	return nil
}

func (k *Runtime) fetchKubeconfig(m net.IP) error {
	logrus.Infof("waiting around 10 seconds for fetch k0s admin.conf")
	// waiting k0s start success.
	time.Sleep(time.Second * TimeForWaitingK0sStart)
	if err := k.infra.CmdAsync(m, nil, "rm -rf .kube/config && mkdir -p /root/.kube && cp /var/lib/k0s/pki/admin.conf /root/.kube/config"); err != nil {
		return err
	}
	return nil
}
