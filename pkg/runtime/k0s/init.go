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
	"path/filepath"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// generateConfigOnMaster0 generate the k0s.yaml to /etc/k0s/k0s.yaml and lead the controller node install
func (k *Runtime) generateConfigOnMaster0(master0 net.IP, registryInfo string) error {
	if err := k.generateK0sConfig(master0); err != nil {
		return fmt.Errorf("failed to generate config: %v", err)
	}
	logrus.Infof("k0s config created under /etc/k0s")

	if err := k.modifyConfigRepo(master0, registryInfo); err != nil {
		return fmt.Errorf("failed to modify config to private repository: %s", err)
	}
	return nil
}

// GenerateCert generate k0s join token for 100 years.
func (k *Runtime) generateJoinToken(master0 net.IP) error {
	workerTokenCreateCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", WorkerRole, DefaultK0sWorkerJoin)
	controllerTokenCreateCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", ControllerRole, DefaultK0sControllerJoin)
	return k.infra.CmdAsync(master0, nil, workerTokenCreateCMD, controllerTokenCreateCMD)
}

func (k *Runtime) generateK0sConfig(master0 net.IP) error {
	mkdirCMD := "mkdir -p /etc/k0s"
	if _, err := k.infra.Cmd(master0, nil, mkdirCMD); err != nil {
		return err
	}

	configCreateCMD := fmt.Sprintf("k0s config create > %s", DefaultK0sConfigPath)
	if _, err := k.infra.Cmd(master0, nil, configCreateCMD); err != nil {
		return err
	}

	return nil
}

func (k *Runtime) modifyConfigRepo(master0 net.IP, registryInfo string) error {
	addRepoCMD := fmt.Sprintf("sed -i '/  images/ a\\    repository: %s' %s", registryInfo, DefaultK0sConfigPath)
	_, err := k.infra.Cmd(master0, nil, addRepoCMD)
	if err != nil {
		return err
	}
	return nil
}

func (k *Runtime) bootstrapMaster0(master0 net.IP) error {
	bootstrapCMD := fmt.Sprintf("k0s install controller -c %s --cri-socket %s", DefaultK0sConfigPath, ExternalCRIAddress)
	if _, err := k.infra.Cmd(master0, nil, bootstrapCMD); err != nil {
		return err
	}

	startSvcCMD := "k0s start"
	if _, err := k.infra.Cmd(master0, nil, startSvcCMD); err != nil {
		return err
	}

	if err := k.WaitK0sReady(master0); err != nil {
		return err
	}
	// fetch kubeconfig
	if _, err := k.infra.Cmd(master0, nil, "rm -rf .kube/config && mkdir -p /root/.kube && cp /var/lib/k0s/pki/admin.conf /root/.kube/config"); err != nil {
		return err
	}
	logrus.Infof("k0s start successfully on master0")
	return nil
}

// initKube prepare install environment.
func (k *Runtime) initKube(hosts []net.IP) error {
	initKubeletCmd := fmt.Sprintf("cd %s && export RegistryURL=%s && bash %s", filepath.Join(k.infra.GetClusterRootfsPath(), "scripts"), k.registryInfo.URL, "init-kube.sh")
	eg, _ := errgroup.WithContext(context.Background())
	for _, h := range hosts {
		host := h
		eg.Go(func() error {
			if err := k.infra.CmdAsync(host, nil, initKubeletCmd); err != nil {
				return fmt.Errorf("failed to init Kubelet Service on (%s): %s", host, err.Error())
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
