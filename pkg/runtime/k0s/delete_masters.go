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

	"github.com/sealerio/sealer/pkg/client/k8s"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (k *Runtime) deleteMasters(masters []net.IP) error {
	if len(masters) == 0 {
		return nil
	}
	eg, _ := errgroup.WithContext(context.Background())
	for _, master := range masters {
		master := master
		eg.Go(func() error {
			logrus.Infof("Start to delete master %s", master)
			if err := k.deleteMaster(master); err != nil {
				return fmt.Errorf("failed to delete master %s: %v", master, err)
			}
			logrus.Infof("Succeeded in deleting master %s", master)
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
	/** To delete a node from k0s cluster, following these steps.
	STEP1: stop k0s service
	STEP2: reset the node with install configuration
	STEP3: remove k0s cluster config generate by k0s under /etc/k0s
	STEP4: remove private registry config in /etc/host
	STEP5: remove bin file such as: kubectl, and remove .kube directory
	STEP6: remove k0s bin file
	STEP7: delete node though k8s client
	*/
	remoteCleanCmd := []string{"k0s stop",
		fmt.Sprintf("k0s reset --cri-socket %s", ExternalCRI),
		"rm -rf /etc/k0s/",
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", SeaHub),
		fmt.Sprintf("sed -i \"/%s/d\" /etc/hosts", k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, k.RegConfig.Domain),
		fmt.Sprintf("rm -rf %s /%s*", DockerCertDir, SeaHub),
		"rm -rf /usr/bin/kube* && rm -rf ~/.kube/",
		"rm -rf /usr/bin/k0s"}

	if err := ssh.CmdAsync(master, remoteCleanCmd...); err != nil {
		return err
	}

	// remove master
	masterExist := len(k.cluster.GetMasterIPList()) > 1
	if !masterExist {
		return errors.New("can not delete the last master")
	}

	hostname, err := k.getHostName(master)
	if err != nil {
		return err
	}
	client, err := k8s.Newk8sClient()
	if err != nil {
		return err
	}
	if err := client.DeleteNode(hostname); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) getHostName(host net.IP) (string, error) {
	client, err := k8s.Newk8sClient()
	if err != nil {
		return "", err
	}
	nodeList, err := client.ListNodes()
	if err != nil {
		return "", err
	}
	var hosts []string
	for _, node := range nodeList.Items {
		hosts = append(hosts, node.GetName())
	}

	hostName, err := k.CmdToString(host, "hostname", "")
	if err != nil {
		return "", err
	}

	var name string
	for _, h := range hosts {
		if strings.TrimSpace(h) == "" {
			continue
		}
		hh := strings.ToLower(h)
		fromH := strings.ToLower(hostName)
		if hh == fromH {
			name = h
			break
		}
	}
	return name, nil
}
