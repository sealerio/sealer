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

package registry

import (
	"fmt"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"net"
)

// Configurator provide registry lifecycle management.
type Configurator interface {
	// Reconcile will start Or Stop registry
	Reconcile() (Driver, error)

	//Upgrade() (Driver, error)
	//Rollback() (Driver, error)
}

type RegistryConfig struct {
	LocalRegistry    *LocalRegistry
	ExternalRegistry *Registry
}

func NewConfigurator(conf RegistryConfig, containerRuntimeInfo containerruntime.Info, infraDriver infradriver.InfraDriver) (Configurator, error) {
	rootfs := infraDriver.GetClusterRootfs()

	var containerRuntimeConfigurator containerruntime.Configurator

	if containerRuntimeInfo.Type == "docker" {
		containerRuntimeConfigurator = containerruntime.NewDockerRuntimeDriver(infraDriver)
	}

	if containerRuntimeInfo.Type == "containerd" {
		containerRuntimeConfigurator = containerruntime.NewContainerdRuntimeDriver(infraDriver)
	}

	if conf.LocalRegistry != nil {
		return &localSingletonConfigurator{
			rootfs:                       rootfs,
			LocalRegistry:                conf.LocalRegistry,
			configFileGenerator:          NewLocalFileGenerator(rootfs),
			containerRuntimeConfigurator: containerRuntimeConfigurator,
			containerRuntimeInfo:         containerRuntimeInfo,
		}, nil
	}

	if conf.ExternalRegistry != nil {
		return &externalConfigurator{Registry: *conf.ExternalRegistry}, nil
	}

	return nil, fmt.Errorf("")
}

type LocalRegistry struct {
	Registry
	DeployHost   net.IP
	DataDir      string   `json:"dataDir,omitempty" yaml:"dataDir,omitempty"`
	InsecureMode bool     `json:"insecure_mode,omitempty" yaml:"insecure_mode,omitempty"`
	Cert         *TLSCert `json:"cert,omitempty" yaml:"cert,omitempty"`
}

type TLSCert struct {
	SubjectAltName *SubjectAltName `json:"subjectAltName,omitempty" yaml:"subjectAltName,omitempty"`
}

type SubjectAltName struct {
	DNSNames []string `json:"dnsNames,omitempty" yaml:"dnsNames,omitempty"`
	IPs      []string `json:"ips,omitempty" yaml:"ips,omitempty"`
}

type Registry struct {
	Domain string        `json:"domain,omitempty" yaml:"domain,omitempty"`
	Port   int           `json:"port,omitempty" yaml:"port,omitempty"`
	Auth   *RegistryAuth `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type RegistryAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}
