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

	"github.com/sealerio/sealer/common"
	osi "github.com/sealerio/sealer/utils/os"
)

func (k *Runtime) init() error {
	pipeline := []func() error{
		k.GenerateConfigOnMaster0,
		k.BootstrapMaster0,
		// TODO: move all these registry operation to the specific registry packages.
		k.GenerateCert,
		k.ApplyRegistryOnMaster0,
		k.GetKubectlAndKubeconfig,
	}

	for _, f := range pipeline {
		if err := f(); err != nil {
			return fmt.Errorf("failed to prepare Master0 env: %v", err)
		}
	}

	return nil
}

// GenerateConfigOnMaster0 generate the k0s.yaml to /etc/k0s/k0s.yaml and lead the controller node install
func (k *Runtime) GenerateConfigOnMaster0() error {
	if err := k.generateK0sConfig(); err != nil {
		return fmt.Errorf("failed to generate config: %v", err)
	}
	if err := k.modifyConfigRepo(); err != nil {
		return fmt.Errorf("failed to modify config to private repository: %s", err)
	}
	return nil
}

// GenerateCert generate the containerd CA for registry TLS and k0s join token for 100 years.
func (k *Runtime) GenerateCert() error {
	if err := k.generateK0sToken(); err != nil {
		return err
	}

	if err := k.GenerateRegistryCert(); err != nil {
		return err
	}
	return k.SendRegistryCert(k.cluster.GetMasterIPList()[:1])
}

func (k *Runtime) generateK0sConfig() error {
	master0IP := k.cluster.GetMaster0IP()
	ssh, err := k.getHostSSHClient(master0IP)
	if err != nil {
		return err
	}

	mkdirCMD := "mkdir -p /etc/k0s"
	if _, err := ssh.Cmd(master0IP, mkdirCMD); err != nil {
		return err
	}

	configCreateCMD := fmt.Sprintf("k0s config create > %s", DefaultK0sConfigPath)
	if _, err := ssh.Cmd(master0IP, configCreateCMD); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) modifyConfigRepo() error {
	master0IP := k.cluster.GetMaster0IP()
	ssh, err := k.getHostSSHClient(master0IP)
	if err != nil {
		return err
	}

	addRepoCMD := fmt.Sprintf("sed -i '/  images/ a\\    repository: %s' %s", k.RegConfig.Domain+":"+k.RegConfig.Port, DefaultK0sConfigPath)
	_, err = ssh.Cmd(master0IP, addRepoCMD)

	return err
}

func (k *Runtime) BootstrapMaster0() error {
	master0IP := k.cluster.GetMaster0IP()
	ssh, err := k.getHostSSHClient(master0IP)
	if err != nil {
		return err
	}
	bootstrapCMD := fmt.Sprintf("k0s install controller -c %s", DefaultK0sConfigPath)
	if _, err := ssh.Cmd(master0IP, bootstrapCMD); err != nil {
		return err
	}
	startSvcCMD := "k0s start"
	if _, err := ssh.Cmd(master0IP, startSvcCMD); err != nil {
		return err
	}
	if err := k.WaitK0sReady(ssh, master0IP); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) generateK0sToken() error {
	master0IP := k.cluster.GetMaster0IP()
	ssh, err := k.getHostSSHClient(master0IP)
	if err != nil {
		return err
	}
	workerTokenCreateCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", WorkerRole, DefaultK0sWorkerJoin)
	controllerTokenCreateCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", ControllerRole, DefaultK0sControllerJoin)
	return ssh.CmdAsync(master0IP, workerTokenCreateCMD, controllerTokenCreateCMD)
}

func (k *Runtime) GetKubectlAndKubeconfig() error {
	if osi.IsFileExist(common.DefaultKubeConfigFile()) {
		return nil
	}
	client, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get ssh client of master0(%s) when get kubectl and kubeconfig: %v", k.cluster.GetMaster0IP(), err)
	}
	return FetchKubeconfigAndGetKubectl(client, k.cluster.GetMaster0IP(), k.getImageMountDir())
}
