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
	"strings"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	RemoteCleanMasterOrNode = `if which k0s; then k0s stop && k0s reset --config %s --cri-socket %s;fi && \
rm -rf /etc/k0s/
`
	RemoveKubeConfig          = "rm -rf /usr/bin/kube* && rm -rf ~/.kube/"
	RemoveK0sBin              = "rm -rf /usr/bin/k0s"
	RemoteRemoveEtcHost       = "sed -i \"/%s/d\" /etc/hosts"
	RemoteRemoveRegistryCerts = "rm -rf " + DockerCertDir + "/%s*"
	KubeDeleteNode            = "kubectl delete node %s"
)

func (k *Runtime) deleteMasters(masters []net.IP) error {
	if len(masters) == 0 {
		return nil
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, master := range masters {
		master := master
		eg.Go(func() error {
			master := master
			logrus.Infof("Start to delete master %s", master)
			if err := k.deleteMaster(master); err != nil {
				logrus.Errorf("failed to delete master %s: %v", master, err)
			} else {
				logrus.Infof("Succeeded in deleting master %s", master)
			}
			return nil
		})
	}
	return eg.Wait()
}

func (k *Runtime) deleteMaster(master net.IP) error {
	ssh, err := k.getHostSSHClient(master)
	if err != nil {
		return fmt.Errorf("failed to delete master: %v", err)
	}
	remoteCleanCmd := []string{fmt.Sprintf(RemoteCleanMasterOrNode, DefaultK0sConfigPath, ExternalCRI),
		fmt.Sprintf(RemoteRemoveEtcHost, SeaHub),
		fmt.Sprintf(RemoteRemoveRegistryCerts, k.RegConfig.Domain),
		fmt.Sprintf(RemoteRemoveRegistryCerts, SeaHub),
		RemoveKubeConfig,
		RemoveK0sBin}

	if err := ssh.CmdAsync(master, remoteCleanCmd...); err != nil {
		return err
	}

	// remove master
	masterIPs := []net.IP{}
	for _, ip := range k.cluster.GetMasterIPList() {
		if !ip.Equal(master) {
			masterIPs = append(masterIPs, ip)
		}
	}

	if len(masterIPs) > 0 {
		hostname, err := k.isHostName(k.cluster.GetMaster0IP(), master)
		if err != nil {
			return err
		}
		master0SSH, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
		if err != nil {
			return fmt.Errorf("failed to get master0 ssh client: %v", err)
		}

		if err := master0SSH.CmdAsync(k.cluster.GetMaster0IP(), fmt.Sprintf(KubeDeleteNode, strings.TrimSpace(hostname))); err != nil {
			return fmt.Errorf("failed to delete node %s: %v", hostname, err)
		}
	}
	return nil
}

func (k *Runtime) isHostName(master, host net.IP) (string, error) {
	hostString, err := k.CmdToString(master, "kubectl get nodes | grep -v NAME  | awk '{print $1}'", ",")
	if err != nil {
		return "", err
	}
	hostName, err := k.CmdToString(host, "hostname", "")
	if err != nil {
		return "", err
	}
	hosts := strings.Split(hostString, ",")
	var name string
	for _, h := range hosts {
		if strings.TrimSpace(h) == "" {
			continue
		} else {
			hh := strings.ToLower(h)
			fromH := strings.ToLower(hostName)
			if hh == fromH {
				name = h
				break
			}
		}
	}
	return name, nil
}
