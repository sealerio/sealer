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
	"github.com/sealerio/sealer/pkg/infradriver"
	"net"
)

type Info struct {
}

// Driver provide external interaction to work with registry.
type Driver interface {
	// UploadContainerImages2Registry :upload src registry filesystem to registry data directory.
	UploadContainerImages2Registry(src string, host net.IP) error
	GetInfo() Info
}

type LocalRegistryDriver struct {
	infraDriver infradriver.InfraDriver
	DataDir     string
}

func (l LocalRegistryDriver) UploadContainerImages2Registry(src string, host net.IP) error {
	return l.infraDriver.Copy(host, src, l.DataDir)
}

func (l LocalRegistryDriver) GetInfo() Info {
	return Info{}
}

func NewLocalRegistryDriver(dataDir string, infraDriver infradriver.InfraDriver) Driver {
	return LocalRegistryDriver{
		infraDriver: infraDriver,
		DataDir:     dataDir,
	}
}
