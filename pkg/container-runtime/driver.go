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
	"github.com/sealerio/sealer/pkg/infradriver"
	"net"
)

type ContainerRuntimeRegistryConfig struct {
	RegistryDeployHost string
	Endpoint           string
	Domain             string
	Port               string
	Username           string
	Password           string
	CaFile             string
	KeyFile            string
	CertFile           string
}

type ContainerRuntimeDriverConfig struct {
	RegistryConfig ContainerRuntimeRegistryConfig
}

//ContainerRuntimeDriver provide configuration Interface for different container runtime.
type ContainerRuntimeDriver interface {
	// ConfigRegistry config registry via each container runtime ip address
	ConfigRegistry(ContainerRuntimeRegistryConfig, []net.IP) error
}

type DockerRuntimeDriver struct {
	ssh infradriver.InfraDriver
}

func (d DockerRuntimeDriver) ConfigRegistry(config ContainerRuntimeRegistryConfig, ips []net.IP) error {
	// add registry ip to "/etc/hosts" :required
	// docker login with username and password: not required
	// copy ca cert to "/etc/docker/certs.d/${domain}:${port}/: not required
	// modify daemon.json: "mirrors": ["${endpoint}"],docker server need to restart if modify daemon.json : not required
}

func (d DockerRuntimeDriver) configHostsFile(registryIP, domain string, ips []net.IP) error {
	// add registry ip to "/etc/hosts"
}

func (d DockerRuntimeDriver) configAuthInfo(username, password string, ips []net.IP) error {
	// docker login with username and password
}

func (d DockerRuntimeDriver) configRegistryCert(domain, port string, caFile string, ips []net.IP) error {
	// copy ca cert to "/etc/docker/certs.d/${domain}:${port}/
}

func (d DockerRuntimeDriver) configDaemon(endpoint string, ips []net.IP) error {
	// modify daemon.json: "mirrors": ["${endpoint}"],docker server need to restart if modify daemon.json
}

func NewDockerRuntimeDriver(ssh infradriver.InfraDriver) ContainerRuntimeDriver {
	return DockerRuntimeDriver{
		ssh: ssh,
	}
}
