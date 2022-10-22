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
	"net"

	"github.com/sealerio/sealer/pkg/imagedistributor"

	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
)

// Configurator provide registry lifecycle management.
type Configurator interface {
	// Launch will start built-in cluster registry component.
	Launch() error
	// Clean will stop built-in cluster registry component.
	Clean() error

	// InstallOn will install registry configuration on each given hosts.
	InstallOn(hosts []net.IP) error

	// UninstallFrom will uninstall registry configuration on each given hosts.
	UninstallFrom(hosts []net.IP) error

	GetDriver() (Driver, error)

	//Upgrade() (Driver, error)
	//Rollback() (Driver, error)
}

type RegConfig struct {
	LocalRegistry    *LocalRegistry
	ExternalRegistry *Registry
}

func NewConfigurator(conf RegConfig, containerRuntimeInfo containerruntime.Info, infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor) (Configurator, error) {
	if conf.LocalRegistry != nil {
		return &localSingletonConfigurator{
			infraDriver:          infraDriver,
			LocalRegistry:        conf.LocalRegistry,
			containerRuntimeInfo: containerRuntimeInfo,
			distributor:          distributor,
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
	Domain string `json:"domain,omitempty" yaml:"domain,omitempty"`
	Port   int    `json:"port,omitempty" yaml:"port,omitempty"`
	Auth   *Auth  `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type Auth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}
