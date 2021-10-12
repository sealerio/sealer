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

package runtime

import (
	"fmt"
	"strings"

	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	drainCommand          = `kubectl drain $(uname -n) --ignore-daemonsets`
	upgradeMaster0Command = `kubeadm upgrade apply %s`
	upgradeOthersCommand  = `kubeadm upgrade node`
	uncordonCommand       = `kubectl uncordon $(uname -n)`
	restartCommand        = `sudo systemctl daemon-reload && sudo systemctl restart kubelet`
)

func (d *Default) upgrade(cluster *v1.Cluster) error {
	var err error
	client, err := ssh.NewSSHClientWithCluster(cluster)
	if err != nil {
		return err
	}
	version, err := getVersionFromImage(cluster.Spec.Image)
	if err != nil {
		return err
	}
	err = upgradeFirstMaster(client, cluster.Spec.Masters.IPList[0], version)
	if err != nil {
		return err
	}
	err = upgradeOtherMasters(client, cluster.Spec.Masters.IPList[1:])
	if err != nil {
		return err
	}
	err = upgradeNodes(client, cluster.Spec.Nodes.IPList)
	if err != nil {
		return err
	}
	return nil
}

func upgradeFirstMaster(client *ssh.Client, IP string, version string) error {
	var CommandsForUpgrade = []string{
		drainCommand,
		fmt.Sprintf(upgradeMaster0Command, version),
		uncordonCommand,
		restartCommand,
	}
	return client.SSH.CmdAsync(IP, CommandsForUpgrade...)
}

func upgradeOtherMasters(client *ssh.Client, IPs []string) error {
	var CommandsForUpgrade = []string{
		drainCommand,
		upgradeOthersCommand,
		uncordonCommand,
		restartCommand,
	}
	var err error
	for _, ip := range IPs {
		err = client.SSH.CmdAsync(ip, CommandsForUpgrade...)
		if err != nil {
			return err
		}
	}
	return err
}

func upgradeNodes(client *ssh.Client, IPs []string) error {
	var CommandsForUpgrade = []string{
		drainCommand,
		upgradeOthersCommand,
		uncordonCommand,
		restartCommand,
	}
	var err error
	for _, ip := range IPs {
		err = client.SSH.CmdAsync(ip, CommandsForUpgrade...)
		if err != nil {
			return err
		}
	}
	return err
}

func getVersionFromImage(image string) (string, error) {
	n := strings.LastIndex(image, `:`)
	if n < 0 {
		return "", nil
	}
	version := image[n+1:]
	// TODO 对version的有效性进行判断
	return version, nil
}
