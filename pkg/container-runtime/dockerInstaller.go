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

package container_runtime

import (
	"fmt"
	"net"

	"github.com/sealerio/sealer/pkg/infradriver"
)

type DockerInstaller struct {
	Info   Info
	rootfs string
	driver infradriver.InfraDriver
}

func (d *DockerInstaller) InstallOn(hosts []net.IP) error {
	RemoteChmod := "cd %s/scripts && chmod +x docker.sh && bash docker.sh %s %s"
	for _, ip := range hosts {
		initCmd := fmt.Sprintf(RemoteChmod, d.rootfs, d.Info.Config.CgroupDriver, d.Info.Config.LimitNofile)
		err := d.driver.CmdAsync(ip, initCmd)
		if err != nil {
			return fmt.Errorf("failed to exec on host %s the install docker command remote: %s", ip, err)
		}
	}
	return nil
}

func (d *DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	CleanCmd := "cd %s/scripts && chmod +x docker-uninstall.sh && bash docker-uninstall.sh"
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, CleanCmd)
		if err != nil {
			return fmt.Errorf("failed to exec on host %s uninstall docker command remote: %s", ip, err)
		}
	}
	return nil
}

func (d DockerInstaller) GetInfo() (Info, error) {
	info := d.Info
	info.CRISocket = DefaultDockerSocket

	return info, nil
}
