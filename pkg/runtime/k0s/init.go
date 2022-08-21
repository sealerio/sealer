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

	yaml2 "gopkg.in/yaml.v2"
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
	k.modifyConfigRepo()
	if err := k.marshalToFile(); err != nil {
		return fmt.Errorf("failed to write k0s config: %v", err)
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
	createCMD := "mkdir -p /etc/k0s && k0s config create"
	bytes, err := ssh.Cmd(master0IP, createCMD)
	if err != nil {
		return err
	}
	return yaml2.Unmarshal(bytes, &k.k0sConfig)
}

func (k *Runtime) modifyConfigRepo() {
	k.k0sConfig.Spec.Images.Repository = k.RegConfig.Domain + ":" + k.RegConfig.Port
}

func (k *Runtime) marshalToFile() error {
	bytes, err := yaml2.Marshal(k.k0sConfig)
	if err != nil {
		return err
	}
	if err = osi.NewAtomicWriter(DefaultK0sConfigPath).WriteFile(bytes); err != nil {
		return err
	}
	return nil
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
	return nil
}

func (k *Runtime) generateK0sToken() error {
	master0IP := k.cluster.GetMaster0IP()
	ssh, err := k.getHostSSHClient(master0IP)
	if err != nil {
		return err
	}
	workerTokenCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", WorkerRole, DefaultK0sWorkerJoin)
	if _, err := ssh.Cmd(master0IP, workerTokenCMD); err != nil {
		return err
	}
	controllerTokenCMD := fmt.Sprintf("k0s token create --role=%s --expiry=876000h > %s", ControllerRole, DefaultK0sControllerJoin)
	if _, err := ssh.Cmd(master0IP, controllerTokenCMD); err != nil {
		return err
	}
	return nil
}

func (k *Runtime) GetKubectlAndKubeconfig() error {
	if osi.IsFileExist(common.DefaultKubeConfigFile()) {
		return nil
	}
	client, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get ssh client of master0(%s) when get kubbectl and kubeconfig: %v", k.cluster.GetMaster0IP(), err)
	}
	return GetKubectlAndKubeconfig(client, k.cluster.GetMaster0IP(), k.getImageMountDir())
}
