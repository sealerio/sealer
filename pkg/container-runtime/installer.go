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

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
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

// Installer interface defines the methods required for installing, getting information, and uninstalling
type Installer interface {
	InstallOn(hosts []net.IP) error

	GetInfo() (Info, error)

	UnInstallFrom(hosts []net.IP) error

	// TODO: Upgrade upgrades the cluster to a newer version
	//Upgrade() (ContainerRuntimeInfo, error)

	// TODO: Rollback rolls back the cluster to a previous version
	//Rollback() (ContainerRuntimeInfo, error)
}

type Info struct {
	v2.ContainerRuntimeConfig
	CgroupDriver   string
	CRISocket      string
	CertsDir       string
	ConfigFilePath string
}

// NewInstaller creates a new Installer based on the specified ContainerRuntimeConfig and InfraDriver
// The returned Installer will be either a DefaultInstaller for Docker or a DefaultInstaller for containerd,
// depending on the value of the ContainerRuntimeConfig.Type field.
func NewInstaller(conf v2.ContainerRuntimeConfig, driver infradriver.InfraDriver) (Installer, error) {
	// Check container runtime type
	switch conf.Type {
	case common.Docker, "":
		// Set container runtime type to Docker if not specified
		conf.Type = common.Docker

		ret := newDefaultInstaller(driver, conf, DefaultDockerCertsDir, DefaultDockerCRISocket, DefaultCgroupDriver, filepath.Join(common.GetHomeDir(), ".docker", DockerConfigFileName))

		return ret, nil
	case common.Containerd:
		ret := newDefaultInstaller(driver, conf, DefaultContainerdCertsDir, DefaultContainerdCRISocket, DefaultCgroupDriver, "")

		return ret, nil
	default:
		return nil, fmt.Errorf("invalid container runtime type: specify docker OR containerd ")
	}
}

// newDefaultInstaller pass NewInstaller creates a new DefaultInstaller object with the specified parameters
func newDefaultInstaller(driver infradriver.InfraDriver, conf v2.ContainerRuntimeConfig, certsDir string, criSocket string, defaultCgroupDriver string, configFile string) *DefaultInstaller {
	ret := &DefaultInstaller{
		rootfs: driver.GetClusterRootfsPath(),
		driver: driver,
		envs:   driver.GetClusterEnv(),
		Info: Info{
			CertsDir:               certsDir,
			CRISocket:              criSocket,
			ContainerRuntimeConfig: conf,
			ConfigFilePath:         configFile,
		},
	}

	// Set Cgroup driver to default value, or use the value from driver environment if provided
	ret.Info.CgroupDriver = defaultCgroupDriver
	if cd, ok := ret.envs[CgroupDriverArg]; ok && cd != nil {
		ret.Info.CgroupDriver = cd.(string)
	}

	return ret
}
