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

package upgrade

import (
	"fmt"

	"github.com/alibaba/sealer/utils/ssh"
)

const (
	aptUpdate      = `apt-get update`
	aptMarkUnhold  = `apt-mark unhold kubeadm kubelet kubectl`
	aptMarkHold    = `apt-mark hold kubeadm kubelet kubectl`
	checkVersion   = `kubeadm version && kubeadm upgrade plan`
	restartKubelet = `sudo systemctl daemon-reload && sudo systemctl restart kubelet`
	installPackage = `apt-get install -y kubeadm=%s-00 kubelet=%s-00 kubectl=%s-00`
)

type debianDistribution struct {
}

func (d debianDistribution) upgradeFirstMaster(client *ssh.Client, IP, version string) {
	hostname, _ := client.SSH.Cmd(IP, "cat /etc/hostname")
	var err error
	err = preUpgrade(client, IP)
	if err != nil {
		return
	}
	var packageManageCmds = []string{
		fmt.Sprintf(installPackage, version, version, version),
		aptMarkHold,
		checkVersion,
	}

	err = client.SSH.CmdAsync(IP, packageManageCmds...)
	if err != nil {
		return
	}
	pullAndTagDockerImage(client, IP, version)
	var upgradeCmds = []string{
		fmt.Sprintf("kubeadm upgrade apply v%s -y", version),
		fmt.Sprintf("kubectl drain %s --ignore-daemonsets", hostname),
		restartKubelet,
		fmt.Sprintf("kubectl uncordon %s", hostname),
	}
	err = client.SSH.CmdAsync(IP, upgradeCmds...)
	if err != nil {
		return
	}
}
func (d debianDistribution) upgradeOtherMaster(client *ssh.Client, IP, version string) {

}
func (d debianDistribution) upgradeNode(client *ssh.Client, IP, version string) {

}

func preUpgrade(client *ssh.Client, IP string) error {
	var cmds = []string{
		aptUpdate,
		`apt-get install -y apt-transport-https ca-certificates curl`,
		`curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://mirrors.aliyun.com/kubernetes/apt/doc/apt-key.gpg`,
		`echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://mirrors.aliyun.com/kubernetes/apt/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list`,
		aptUpdate,
		aptMarkUnhold,
		aptUpdate,
	}
	return client.SSH.CmdAsync(IP, cmds...)
}
