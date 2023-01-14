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
	v2 "github.com/sealerio/sealer/types/api/v2"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/common"

	"github.com/sealerio/sealer/pkg/infradriver"
)

const (
	DefaultDockerCRISocket     = "/var/run/dockershim.sock"
	DefaultCgroupDriver        = "systemd"
	DefaultDockerCertsDir      = "/etc/docker/certs.d"
	DefaultContainerdCRISocket = "/run/containerd/containerd.sock"
	DefaultContainerdCertsDir  = "/etc/containerd/certs.d"
	DockerConfigFileName       = "config.json"
)

const (
	CgroupDriverArg = "CgroupDriver"
)

type Installer interface {
	InstallOn(hosts []net.IP) error

	GetInfo() (Info, error)

	UnInstallFrom(hosts []net.IP) error

	//Upgrade() (ContainerRuntimeInfo, error)
	//Rollback() (ContainerRuntimeInfo, error)
}

type Info struct {
	v2.ContainerRuntimeConfig
	CgroupDriver   string
	CRISocket      string
	CertsDir       string
	ConfigFilePath string
}

func NewInstaller(conf v2.ContainerRuntimeConfig, driver infradriver.InfraDriver) (Installer, error) {
	switch conf.Type {
	case "docker", "":
		ret := &DefaultInstaller{
			rootfs: driver.GetClusterRootfsPath(),
			driver: driver,
			envs:   driver.GetClusterEnv(),
			Info: Info{
				CertsDir:               DefaultDockerCertsDir,
				CRISocket:              DefaultDockerCRISocket,
				ContainerRuntimeConfig: conf,
				ConfigFilePath:         filepath.Join(common.GetHomeDir(), ".docker", DockerConfigFileName),
			},
		}
		ret.Info.CgroupDriver = DefaultCgroupDriver
		if cd, ok := ret.envs[CgroupDriverArg]; ok && cd != nil {
			ret.Info.CgroupDriver = cd.(string)
		}

		return ret, nil
	case "containerd":
		ret := &DefaultInstaller{
			rootfs: driver.GetClusterRootfsPath(),
			driver: driver,
			envs:   driver.GetClusterEnv(),
			Info: Info{
				CertsDir:               DefaultContainerdCertsDir,
				CRISocket:              DefaultContainerdCRISocket,
				ContainerRuntimeConfig: conf,
			},
		}
		ret.Info.CgroupDriver = DefaultCgroupDriver
		if cd, ok := ret.envs[CgroupDriverArg]; ok && cd != nil {
			ret.Info.CgroupDriver = cd.(string)
		}

		return ret, nil
	default:
		return nil, fmt.Errorf("please enter the correct container type")
	}
}
