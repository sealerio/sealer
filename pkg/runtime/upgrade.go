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
	"path/filepath"
	"strings"
)

const (
	chmodCmd       = `chmod +x %s/*`
	mvCmd          = `mv %s/* /usr/bin`
	getNodeNameCmd = `$(uname -n | tr '[A-Z]' '[a-z]')`
	drainCmd       = `kubectl drain ` + getNodeNameCmd + ` --ignore-daemonsets`
	upgradeCmd     = `kubeadm upgrade %s`
	restartCmd     = `systemctl daemon-reload && systemctl restart kubelet`
	uncordonCmd    = `kubectl uncordon ` + getNodeNameCmd
)

func (k *KubeadmRuntime) upgrade() error {
	var err error
	binpath := filepath.Join(k.getRootfs(), `bin`)

	err = k.upgradeFirstMaster(k.getMaster0IP(), binpath, k.getKubeVersion())
	if err != nil {
		return err
	}
	err = k.upgradeOtherMasters(k.getMasterIPList()[1:], binpath)
	if err != nil {
		return err
	}
	err = k.upgradeNodes(k.getNodesIPList(), binpath)
	if err != nil {
		return err
	}
	return nil
}

func (k *KubeadmRuntime) upgradeFirstMaster(IP string, binpath, version string) error {
	var firstMasterCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		drainCmd,
		fmt.Sprintf(upgradeCmd, strings.Join([]string{`apply`, version, `-y`}, " ")),
		restartCmd,
		uncordonCmd,
	}
	ssh, err := k.getHostSSHClient(IP)
	if err != nil {
		return fmt.Errorf("upgrade master0 failed %v", err)
	}
	return ssh.CmdAsync(IP, firstMasterCmds...)
}

func (k *KubeadmRuntime) upgradeOtherMasters(IPs []string, binpath string) error {
	var otherMasterCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		drainCmd,
		fmt.Sprintf(upgradeCmd, `node`),
		restartCmd,
		uncordonCmd,
	}
	var err error
	for _, ip := range IPs {
		ssh, err := k.getHostSSHClient(ip)
		if err != nil {
			return fmt.Errorf("upgrade other masters failed: %v", err)
		}
		err = ssh.CmdAsync(ip, otherMasterCmds...)
		if err != nil {
			return err
		}
	}
	return err
}

func (k *KubeadmRuntime) upgradeNodes(IPs []string, binpath string) error {
	var nodeCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		fmt.Sprintf(upgradeCmd, `node`),
		restartCmd,
	}
	var err error
	for _, ip := range IPs {
		ssh, err := k.getHostSSHClient(ip)
		if err != nil {
			return fmt.Errorf("upgrade node failed: %v", err)
		}
		err = ssh.CmdAsync(ip, nodeCmds...)
		if err != nil {
			return err
		}
	}
	return err
}
