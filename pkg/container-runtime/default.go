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

package containerruntime

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/pkg/infradriver"
)

type DefaultInstaller struct {
	Info
	envs   map[string]string
	rootfs string
	driver infradriver.InfraDriver
}

func (d *DefaultInstaller) InstallOn(hosts []net.IP) error {
	installCmd := fmt.Sprintf("bash %s", filepath.Join(d.rootfs, "scripts", d.getInstallScriptName()))
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, d.envs, installCmd)
		if err != nil {
			return fmt.Errorf("failed to install %s: execute command(%s) on host (%s): error(%v)", d.Type, installCmd, ip, err)
		}
	}
	return nil
}

func (d *DefaultInstaller) UnInstallFrom(hosts []net.IP) error {
	cleanCmd := fmt.Sprintf("bash %s", filepath.Join(d.rootfs, "scripts", d.getUnInstallScriptName()))
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, d.envs, cleanCmd)
		if err != nil {
			return fmt.Errorf("failed to uninstall %s: execute command(%s) on host (%s): error(%v)", d.Type, cleanCmd, ip, err)
		}
	}
	return nil
}

func (d *DefaultInstaller) GetInfo() (Info, error) {
	return d.Info, nil
}

func (d *DefaultInstaller) getInstallScriptName() string {
	return fmt.Sprintf("%s.sh", d.Type)
}

func (d *DefaultInstaller) getUnInstallScriptName() string {
	return fmt.Sprintf("uninstall-%s.sh", d.Type)
}
