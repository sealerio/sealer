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

	"github.com/sealerio/sealer/pkg/imagedistributor"
)

type Info struct {
	URL string
}

// Driver provide external interaction to work with registry.
type Driver interface {
	// UploadContainerImages2Registry :upload src registry filesystem to registry data directory.
	UploadContainerImages2Registry() error
	GetInfo() Info
}

type localRegistryDriver struct {
	dataDir     string
	endpoint    string
	distributor imagedistributor.Distributor
	deployHosts []net.IP
}

func (l localRegistryDriver) UploadContainerImages2Registry() error {
	return l.distributor.DistributeRegistry(l.deployHosts, l.dataDir)
}

func (l localRegistryDriver) GetInfo() Info {
	return Info{URL: l.endpoint}
}

func newLocalRegistryDriver(endpoint string, dataDir string, deployHosts []net.IP, distributor imagedistributor.Distributor) Driver {
	return localRegistryDriver{
		endpoint:    endpoint,
		distributor: distributor,
		dataDir:     dataDir,
		deployHosts: deployHosts,
	}
}

type externalRegistryDriver struct {
	endpoint string
}

func (l externalRegistryDriver) UploadContainerImages2Registry() error {
	//not implement currently
	return nil
}

func (l externalRegistryDriver) GetInfo() Info {
	return Info{URL: l.endpoint}
}

func newExternalRegistryDriver(endpoint string) Driver {
	return externalRegistryDriver{
		endpoint: endpoint,
	}
}
