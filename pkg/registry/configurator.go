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

func NewConfigurator(containerRuntimeInfo containerruntime.Info, infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor) (Configurator, error) {
	conf := infraDriver.GetClusterRegistryConfig()
	if conf.LocalRegistry != nil {
		return &localSingletonConfigurator{
			infraDriver:          infraDriver,
			LocalRegistry:        conf.LocalRegistry,
			containerRuntimeInfo: containerRuntimeInfo,
			distributor:          distributor,
		}, nil
	}

	if conf.ExternalRegistry != nil {
		return &externalConfigurator{RegistryConfig: conf.ExternalRegistry.RegistryConfig}, nil
	}

	return nil, fmt.Errorf("")
}
