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

package clusterruntime

import (
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
)

func (i *Installer) Rollback() error {
	var (
		masters = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0 = masters[0]
		workers = getWorkerIPList(i.infraDriver)
		all     = append(masters, workers...)
		rootfs  = i.infraDriver.GetClusterRootfsPath()
	)

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return err
	}

	var deployHosts []net.IP
	if i.regConfig.LocalRegistry != nil {
		installer := registry.NewInstaller(nil, i.regConfig.LocalRegistry, i.infraDriver, i.Distributor)
		if *i.regConfig.LocalRegistry.HA {
			deployHosts, err = installer.Reconcile(masters)
			if err != nil {
				return err
			}
		} else {
			deployHosts, err = installer.Reconcile([]net.IP{master0})
			if err != nil {
				return err
			}
		}
	}
	registryConfigurator, err := registry.NewConfigurator(deployHosts, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	if err = registryConfigurator.InstallOn(masters, workers); err != nil {
		return err
	}

	if err := i.runClusterHook(master0, RollbackCluster); err != nil {
		return err
	}

	//distribute rootfs after rollback
	if err := i.Distributor.Distribute(all, rootfs); err != nil {
		return err
	}

	return nil
}
