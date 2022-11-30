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
	"net"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

type externalConfigurator struct {
	v2.RegistryConfig
}

func (c *externalConfigurator) Launch() error {
	panic("implement external")
}

func (c *externalConfigurator) Clean() error {
	panic("implement external")
}

func (c *externalConfigurator) InstallOn(hosts []net.IP) error {
	panic("implement external")
}

func (c *externalConfigurator) UninstallFrom(hosts []net.IP) error {
	panic("implement external")
}

func (c *externalConfigurator) GetDriver() (Driver, error) {
	panic("implement external")
}
