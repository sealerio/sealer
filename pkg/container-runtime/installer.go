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
	"net"

	"github.com/sealerio/sealer/pkg/infradriver"
)

const (
	DefaultDockerCRISocket     = "/var/run/dockershim.sock"
	DefaultContainerdCRISocket = "/run/containerd/containerd.sock"
	DefaultSystemdDriver       = "systemd"
	DefaultCgroupfsDriver      = "cgroupfs"
	Docker                     = "docker"
	RemoteChmod                = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh /var/lib/docker %s %s %s %s"
	CleanCmd                   = "cd %s  && chmod +x scripts/* && cd scripts && bash clean.sh"
	ContainerdRemoteChmod      = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh %s %s"
	DefaultLimitNoFile         = "infinity"
	Containerd                 = "containerd"
	DefaultDomain              = "sea.hub"
	DefaultPort                = "5000"
)

// 容器运行时安装器
type Installer interface {
	InstallOn(hosts []net.IP) (*Info, error)

	UnInstallFrom(hosts []net.IP) error

	//Upgrade() (ContainerRuntimeInfo, error)
	//Rollback() (ContainerRuntimeInfo, error)
}

type Config struct {
	Type         string
	LimitNofile  string `json:"limitNofile,omitempty" yaml:"limitNofile,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty" yaml:"cgroupDriver,omitempty"`
}

type Info struct {
	Config    Config
	CRISocket string
}

func NewInstaller(conf Config, rootfs string, driver infradriver.InfraDriver) Installer {
	config := &DockerInstaller{
		Info: Info{
			Config: conf,
		},
		rootfs: rootfs,
		driver: driver,
	}
	return config
}
