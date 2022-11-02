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

type DockerInstaller struct {
	Info
	rootfs string
	driver infradriver.InfraDriver
}

func (d *DockerInstaller) InstallOn(hosts []net.IP) error {
	installCmd := fmt.Sprintf("bash %s %s", filepath.Join(d.rootfs, "scripts", "docker.sh"), d.Info.LimitNofile)
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, installCmd)
		if err != nil {
			return fmt.Errorf("failed to install docker: execute command(%s) on host (%s): error(%v)", installCmd, ip, err)
		}
	}
	return nil
}

func (d *DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	cleanCmd := fmt.Sprintf("if ! which docker;then exit 0;fi; bash %s", filepath.Join(d.rootfs, "scripts", "uninstall-docker.sh"))
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, cleanCmd)
		if err != nil {
			return fmt.Errorf("failed to uninstall docker: execute command(%s) on host (%s): error(%v)", cleanCmd, ip, err)
		}
	}
	return nil
}

func (d DockerInstaller) GetInfo() (Info, error) {
	return d.Info, nil
}
