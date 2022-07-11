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
	"net"
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
	binPath := filepath.Join(k.getRootfs(), `bin`)

	err = k.upgradeFirstMaster(k.GetMaster0IP(), binPath, k.getKubeVersion())
	if err != nil {
		return err
	}
	err = k.upgradeOtherMasters(k.GetMasterIPList()[1:], binPath, k.getKubeVersion())
	if err != nil {
		return err
	}
	err = k.upgradeNodes(k.GetNodeIPList(), binPath)
	if err != nil {
		return err
	}
	return nil
}

func (k *KubeadmRuntime) upgradeFirstMaster(IP net.IP, binPath, version string) error {
	var drain string
	//if version >= 1.20.x,add flag `--delete-emptydir-data`
	if VersionCompare(version, V1200) {
		drain = fmt.Sprintf("%s %s", drainCmd, "--delete-emptydir-data")
	} else {
		drain = fmt.Sprintf("%s %s", drainCmd, "--delete-local-data")
	}

	var firstMasterCmds = []string{
		fmt.Sprintf(chmodCmd, binPath),
		fmt.Sprintf(mvCmd, binPath),
		drain,
		fmt.Sprintf(upgradeCmd, strings.Join([]string{`apply`, version, `-y`}, " ")),
		restartCmd,
		uncordonCmd,
	}
	ssh, err := k.getHostSSHClient(IP)
	if err != nil {
		return fmt.Errorf("failed to get master0 ssh client: %v", err)
	}
	return ssh.CmdAsync(IP, firstMasterCmds...)
}

func (k *KubeadmRuntime) upgradeOtherMasters(IPs []net.IP, binpath, version string) error {
	var drain string
	//if version >= 1.20.x,add flag `--delete-emptydir-data`
	if VersionCompare(version, V1200) {
		drain = fmt.Sprintf("%s %s", drainCmd, "--delete-emptydir-data")
	} else {
		drain = fmt.Sprintf("%s %s", drainCmd, "--delete-local-data")
	}

	var otherMasterCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		drain,
		fmt.Sprintf(upgradeCmd, `node`),
		restartCmd,
		uncordonCmd,
	}
	var err error
	for _, ip := range IPs {
		ssh, err := k.getHostSSHClient(ip)
		if err != nil {
			return fmt.Errorf("failed to get ssh client of host(%s): %v", ip, err)
		}
		err = ssh.CmdAsync(ip, otherMasterCmds...)
		if err != nil {
			return err
		}
	}
	return err
}

func (k *KubeadmRuntime) upgradeNodes(IPs []net.IP, binpath string) error {
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
			return fmt.Errorf("failed to get ssh client of host(%s): %v", ip, err)
		}
		err = ssh.CmdAsync(ip, nodeCmds...)
		if err != nil {
			return err
		}
	}
	return err
}
