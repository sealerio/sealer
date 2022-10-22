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
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
)

func (i *Installer) ScaleUp(newMasters, newWorkers []net.IP) (registry.Driver, runtime.Driver, error) {
	master0 := i.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	all := append(newMasters, newWorkers...)

	// distribute rootfs
	if err := i.Distributor.DistributeRootfs(all, i.infraDriver.GetClusterRootfsPath()); err != nil {
		return nil, nil, err
	}

	// set HostAlias
	if err := i.infraDriver.SetClusterHostAliases(all); err != nil {
		return nil, nil, err
	}

	if err := i.runClusterHook(master0, PreScaleUpCluster); err != nil {
		return nil, nil, err
	}

	if err := i.runHostHook(PreInitHost, all); err != nil {
		return nil, nil, err
	}

	if err := i.containerRuntimeInstaller.InstallOn(all); err != nil {
		return nil, nil, err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
	if err != nil {
		return nil, nil, err
	}

	if err := registryConfigurator.InstallOn(all); err != nil {
		return nil, nil, err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return nil, nil, err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, crInfo, registryDriver.GetInfo())
	if err != nil {
		return nil, nil, err
	}

	if err := kubeRuntimeInstaller.ScaleUp(newMasters, newWorkers); err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if err := i.runHostHook(PostInitHost, all); err != nil {
		return nil, nil, err
	}

	if err := i.runClusterHook(master0, PostScaleUpCluster); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) ScaleDown(mastersToDelete, workersToDelete []net.IP) (registry.Driver, runtime.Driver, error) {
	if len(workersToDelete) > 0 {
		if err := confirmDeleteHosts(common.NODE, workersToDelete); err != nil {
			return nil, nil, err
		}
	}

	if len(mastersToDelete) > 0 {
		if err := confirmDeleteHosts(common.MASTER, mastersToDelete); err != nil {
			return nil, nil, err
		}
	}

	all := append(mastersToDelete, workersToDelete...)
	// delete HostAlias
	if err := i.infraDriver.DeleteClusterHostAliases(all); err != nil {
		return nil, nil, err
	}

	if err := i.runHostHook(PreCleanHost, all); err != nil {
		return nil, nil, err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
	if err != nil {
		return nil, nil, err
	}

	if err = registryConfigurator.UninstallFrom(all); err != nil {
		return nil, nil, err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return nil, nil, err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, crInfo, registryDriver.GetInfo())
	if err != nil {
		return nil, nil, err
	}

	if err = kubeRuntimeInstaller.ScaleDown(mastersToDelete, workersToDelete); err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if err = i.containerRuntimeInstaller.UnInstallFrom(all); err != nil {
		return nil, nil, err
	}

	if err = i.runHostHook(PostCleanHost, all); err != nil {
		return nil, nil, err
	}

	if err = i.Distributor.Restore(i.infraDriver.GetClusterBasePath(), all); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}
