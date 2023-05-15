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
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/registry"
)

func (i *Installer) UnInstall() error {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	master0 := masters[0]
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)

	_, err := CheckNodeSSH(i.infraDriver, all)
	if err != nil {
		return err
	}

	if err := i.runClusterHook(master0, PreUnInstallCluster); err != nil {
		return err
	}

	if err := i.runHostHook(PreCleanHost, all); err != nil {
		return err
	}

	kubeRuntimeInstaller, err := getClusterRuntimeInstaller(i.clusterRuntimeType, i.infraDriver,
		containerruntime.Info{}, registry.Info{}, i.KubeadmConfig)
	if err != nil {
		return err
	}

	if err = kubeRuntimeInstaller.Reset(); err != nil {
		return err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return err
	}

	if i.regConfig.LocalRegistry != nil {
		if *i.regConfig.LocalRegistry.HA {
			installer := registry.NewInstaller(masters, i.regConfig.LocalRegistry, i.infraDriver, i.Distributor)
			err = installer.Clean()
			if err != nil {
				return err
			}
		}

		installer := registry.NewInstaller([]net.IP{master0}, i.regConfig.LocalRegistry, i.infraDriver, i.Distributor)
		err = installer.Clean()
		if err != nil {
			return err
		}
	}

	registryConfigurator, err := registry.NewConfigurator(nil, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	if err = registryConfigurator.UninstallFrom(masters, workers); err != nil {
		return err
	}

	if err = i.containerRuntimeInstaller.UnInstallFrom(all); err != nil {
		return err
	}

	if err = i.runHostHook(PostCleanHost, all); err != nil {
		return err
	}

	if err = i.runClusterHook(master0, PostUnInstallCluster); err != nil {
		return err
	}

	// delete HostAlias
	if err := i.infraDriver.DeleteClusterHostAliases(all); err != nil {
		return err
	}

	if err = i.Distributor.Restore(i.infraDriver.GetClusterBasePath(), all); err != nil {
		return err
	}

	return nil
}
