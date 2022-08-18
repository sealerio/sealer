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
	"github.com/sealerio/sealer/pkg/infradriver"
	"path/filepath"
	"strings"
)

const (
	DefaultEndpoint = "sea.hub:5000"
)

//Configurator provide configuration Interface for different container runtime.
type Configurator interface {
	// ConfigDaemonService config container runtime daemon service
	ConfigDaemonService(DaemonConfig) error
}

type DaemonConfig struct {
	// Endpoint typically is "${domain}+${port}"
	Endpoint string
}

type DockerRuntimeConfigurator struct {
	infraDriver infradriver.InfraDriver
}

func (d DockerRuntimeConfigurator) ConfigDaemonService(config DaemonConfig) error {
	var configDaemonCmd []string

	cmd := d.configRegistryEndpoint(config.Endpoint)

	// no need to reconfigure docker daemon service if cmd is nil
	if cmd == "" {
		return nil
	}

	configDaemonCmd = append(configDaemonCmd, cmd)
	configDaemonCmd = append(configDaemonCmd, "systemctl daemon-reload")

	for _, ip := range d.infraDriver.GetHostIPList() {
		host := ip
		err := d.infraDriver.CmdAsync(host, strings.Join(configDaemonCmd, " && "))
		if err != nil {
			return err
		}
	}

	return nil
}

func (d DockerRuntimeConfigurator) configRegistryEndpoint(endpoint string) string {
	// modify daemon.json: "mirrors": ["${endpoint}"],docker server need to restart if modify daemon.json
	if endpoint == DefaultEndpoint {
		return ""
	}

	return fmt.Sprintf("sed -i \"s/sea.hub:5000/%s/g\" /etc/docker/daemon.json", endpoint)
}

func NewDockerRuntimeDriver(infraDriver infradriver.InfraDriver) Configurator {
	return DockerRuntimeConfigurator{
		infraDriver: infraDriver,
	}
}

type ContainerdRuntimeConfigurator struct {
	rootfs      string
	infraDriver infradriver.InfraDriver
}

func (c ContainerdRuntimeConfigurator) ConfigDaemonService(config DaemonConfig) error {
	var configDaemonCmd []string

	cmd := c.configRegistryEndpoint(config.Endpoint)

	// no need to reconfigure containerd daemon service if cmd is nil
	if cmd == nil {
		return nil
	}

	configDaemonCmd = append(configDaemonCmd, cmd...)
	configDaemonCmd = append(configDaemonCmd, "systemctl daemon-reload")

	for _, ip := range c.infraDriver.GetHostIPList() {
		host := ip
		err := c.infraDriver.CmdAsync(host, strings.Join(configDaemonCmd, " && "))
		if err != nil {
			return err
		}
	}

	return nil
}

func (c ContainerdRuntimeConfigurator) configRegistryEndpoint(endpoint string) []string {
	if endpoint == DefaultEndpoint {
		return nil
	}

	configContainerdCmd := []string{}
	configFile := filepath.Join(c.rootfs, "etc/dump-config.toml")

	// containerd --config ${dump-config.toml} config dump > /etc/containerd/config.toml
	configContainerdCmd = append(configContainerdCmd,
		fmt.Sprintf("sed -i \"s/sea.hub:5000/%s/g\" %s",
			configFile, endpoint))

	configContainerdCmd = append(configContainerdCmd,
		fmt.Sprintf("containerd --config %s config dump > /etc/containerd/config.toml", configFile))

	return configContainerdCmd
}

func NewContainerdRuntimeDriver(infraDriver infradriver.InfraDriver) Configurator {
	return ContainerdRuntimeConfigurator{
		rootfs:      infraDriver.GetClusterRootfs(),
		infraDriver: infraDriver,
	}
}
