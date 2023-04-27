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

	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type LocalRegistryInfo struct {
	*v2.LocalRegistry
	DeployHosts []net.IP `json:"deployHosts,omitempty"`
	Vip         string   `json:"vip,omitempty"`
}

type RegistryInfo struct { // nolint
	External *v2.ExternalRegistry `json:"external,omitempty"`
	Local    LocalRegistryInfo    `json:"local,omitempty"`
}

// Configurator provide registry configuration management
type Configurator interface {
	// InstallOn will install registry configuration on each given hosts.
	InstallOn(masters, nodes []net.IP) error

	// UninstallFrom will uninstall registry configuration on each given hosts.
	UninstallFrom(masters, nodes []net.IP) error

	GetDriver() (Driver, error)

	GetRegistryInfo() RegistryInfo
}

func NewConfigurator(deployHosts []net.IP,
	containerRuntimeInfo containerruntime.Info,
	regConfig v2.Registry,
	infraDriver infradriver.InfraDriver,
	distributor imagedistributor.Distributor) (Configurator, error) {
	if regConfig.LocalRegistry != nil {
		return &localConfigurator{
			deployHosts:          deployHosts,
			infraDriver:          infraDriver,
			LocalRegistry:        regConfig.LocalRegistry,
			containerRuntimeInfo: containerRuntimeInfo,
			distributor:          distributor,
		}, nil
	}

	if regConfig.ExternalRegistry != nil {
		return NewExternalConfigurator(
			regConfig.ExternalRegistry,
			containerRuntimeInfo,
			infraDriver,
		)
	}

	return nil, fmt.Errorf("unsupported registry type")
}
