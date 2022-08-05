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
)

func (k *Runtime) init() error {
	pipeline := []func() error{
		k.MergeConfigOnMaster0,
		k.SSHKeyGenAndCopyIDToHosts,
		// TODO: move all these registry operation to the specific registry packages.
		k.GenerateCert,
		k.ApplyRegistryOnMaster0,
	}

	for _, f := range pipeline {
		if err := f(); err != nil {
			return fmt.Errorf("failed to prepare Master0 env: %v", err)
		}
	}

	return nil
}

// MergeConfigOnMaster0 convert the cluster file spec to kubectl.yaml (k0sctl config file) to lead cluster run
func (k *Runtime) MergeConfigOnMaster0() error {
	if err := k.K0sConfig.ConvertTok0sConfig(k.cluster); err != nil {
		return fmt.Errorf("failed to convert to k0s config from clusterfile: %v", err)
	}
	ssh, err := k.getHostSSHClient(k.cluster.GetMaster0IP())
	if err != nil {
		return fmt.Errorf("failed to get ssh client: %v", err)
	}
	output, err := ssh.Cmd(k.cluster.GetMaster0IP(), GetK0sVersionCMD)
	if err != nil {
		return err
	}

	k.K0sConfig.DefineConfigFork0s(string(output), k.RegConfig.Domain, k.RegConfig.Port, k.cluster.Name)

	//write k0sctl.yaml to master0 rootfs
	if err := k.K0sConfig.WriteConfigToMaster0(k.getRootfs()); err != nil {
		return err
	}
	return nil
}

// SSHKeyGenAndCopyIDToHosts use ssh-copy-id to prepare a no password ssh-client for k0sctl install environment.
func (k *Runtime) SSHKeyGenAndCopyIDToHosts() error {
	return k.sshKeyGenAndCopyIDToHosts()
}

// GenerateCert generate the containerd CA for registry TLS.
func (k *Runtime) GenerateCert() error {
	if err := k.GenerateRegistryCert(); err != nil {
		return err
	}
	return k.SendRegistryCert(k.cluster.GetMasterIPList()[:1])
}

// sshKeyGenAndCopyIDToHosts use ssh-key-gen and ssh-copy-id to prepare an env without password for k0sctl.
func (k *Runtime) sshKeyGenAndCopyIDToHosts() error {
	if err := k.sshKeyGen(); err != nil {
		return err
	}
	return k.sshCopyIDToEveryHost()
}
