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

func (d *DockerInstaller) InstallOn(hosts []net.IP) (*Info, error) {
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		initCmd := fmt.Sprintf(RemoteChmod, d.rootfs, DefaultDomain, DefaultPort, d.Info.Config.CgroupDriver, d.Info.Config.LimitNofile)
		err := d.driver.CmdAsync(IP, initCmd)
		if err != nil {
			return nil, fmt.Errorf("failed to remote exec init cmd: %s", err)
		}
	}
	return &d.Info, nil
}

func (d *DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		err := d.driver.CmdAsync(IP, CleanCmd)
		if err != nil {
			return fmt.Errorf("failed to remote exec clean cmd: %s", err)
		}
	}
	return nil
}
