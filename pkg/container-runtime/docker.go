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

	"github.com/sealerio/sealer/pkg/infradriver"
)

type DockerInstaller struct {
	Info
	rootfs string
	driver infradriver.InfraDriver
}

func (d *DockerInstaller) InstallOn(hosts []net.IP) error {
	RemoteChmod := "cd %s/scripts && chmod +x docker.sh && bash docker.sh %s %s"
	for _, ip := range hosts {
		initCmd := fmt.Sprintf(RemoteChmod, d.rootfs, d.Info.CgroupDriver, d.Info.LimitNofile)
		err := d.driver.CmdAsync(ip, initCmd)
		if err != nil {
			return fmt.Errorf("failed to execute install command(%s) on host (%s): error(%v)", initCmd, ip, err)
		}
	}
	return nil
}

func (d *DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	//todo need to cooperator with the rootfs files, so the name of uninstall bash file need to discuss
	cleanCmd := fmt.Sprintf("cd %s/scripts && chmod +x uninstall-docker.sh && bash uninstall-docker.sh", d.driver.GetClusterRootfs())
	for _, ip := range hosts {
		err := d.driver.CmdAsync(ip, cleanCmd)
		if err != nil {
			return fmt.Errorf("failed to execute clean command(%s) on host (%s): error(%v)", cleanCmd, ip, err)
		}
	}
	return nil
}

func (d DockerInstaller) GetInfo() (Info, error) {
	info := d.Info
	info.CRISocket = DefaultDockerSocket
	info.CertsDir = DockerCertsDir

	return info, nil
}
