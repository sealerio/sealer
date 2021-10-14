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

	"regexp"
	"strings"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils/ssh"
)

const (
	chmodCmd    = `chmod +x %s/*`
	mvCmd       = `mv %s/* /usr/bin`
	drainCmd    = `kubectl drain $(uname -n) --ignore-daemonsets`
	upgradeCmd  = `kubeadm upgrade %s`
	restartCmd  = `sudo systemctl daemon-reload && sudo systemctl restart kubelet`
	uncordonCmd = `kubectl uncordon $(uname -n)`
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
	binpath := filepath.Join(common.DefaultTheClusterRootfsDir(cluster.Name), `bin`)

	err = upgradeFirstMaster(client, cluster.Spec.Masters.IPList[0], binpath, version)
	if err != nil {
		return err
	}
	err = upgradeOtherMasters(client, cluster.Spec.Masters.IPList[1:], binpath)
	if err != nil {
		return err
	}
	err = upgradeNodes(client, cluster.Spec.Nodes.IPList, binpath)
	if err != nil {
		return err
	}
	return nil
}

func upgradeFirstMaster(client *ssh.Client, IP string, binpath, version string) error {
	var firstMasterCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		drainCmd,
		fmt.Sprintf(upgradeCmd, strings.Join([]string{`apply`, version, `-y`}, " ")),
		restartCmd,
		uncordonCmd,
	}
	return client.SSH.CmdAsync(IP, firstMasterCmds...)
}

func upgradeOtherMasters(client *ssh.Client, IPs []string, binpath string) error {
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
		err = client.SSH.CmdAsync(ip, otherMasterCmds...)
		if err != nil {
			return err
		}
	}
	return err
}

func upgradeNodes(client *ssh.Client, IPs []string, binpath string) error {
	var nodeCmds = []string{
		fmt.Sprintf(chmodCmd, binpath),
		fmt.Sprintf(mvCmd, binpath),
		fmt.Sprintf(upgradeCmd, `node`),
		restartCmd,
	}
	var err error
	for _, ip := range IPs {
		err = client.SSH.CmdAsync(ip, nodeCmds...)
		if err != nil {
			return err
		}
	}
	return err
}

func getVersionFromImage(image string) (string, error) {
	n := strings.LastIndex(image, `:`)
	if n < 0 {
		return "", fmt.Errorf("upgrade should have a version")
	}
	version := image[n+1:]
	re := regexp.MustCompile(`v[0-9]+.[0-9]+.[0-9]+`) //eg:v1.19.10
	version = re.FindString(version)
	if version == "" {
		return version, fmt.Errorf("invalid version,please check your input and retry")
	}
	return version, nil
}
