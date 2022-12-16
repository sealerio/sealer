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

	netutils "github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
)

func (i *Installer) ScaleUp(newMasters, newWorkers []net.IP) (registry.Driver, runtime.Driver, error) {
	rootfs := i.infraDriver.GetClusterRootfsPath()
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	master0 := masters[0]
	registryDeployHosts := []net.IP{master0}
	all := append(newMasters, newWorkers...)

	// set HostAlias
	if err := i.infraDriver.SetClusterHostAliases(all); err != nil {
		return nil, nil, err
	}
	// distribute rootfs
	if err := i.Distributor.Distribute(all, rootfs); err != nil {
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

	// reconcile registry node if local registry is ha mode.
	if i.regConfig.LocalRegistry != nil && i.regConfig.LocalRegistry.HaMode {
		registryDeployHosts, err = registry.NewInstaller(netutils.RemoveIPs(masters, newMasters), i.regConfig.LocalRegistry, i.infraDriver, i.Distributor).Reconcile(masters)
		if err != nil {
			return nil, nil, err
		}
	}

	registryConfigurator, err := registry.NewConfigurator(registryDeployHosts, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
		return nil, nil, err
	}

	if err = registryConfigurator.InstallOn(newMasters, newWorkers); err != nil {
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

	if err := i.runHostHook(PostInitHost, all); err != nil {
		return nil, nil, err
	}

	if err := i.runClusterHook(master0, PostScaleUpCluster); err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if err := i.setNodeLabels(all, runtimeDriver); err != nil {
		return nil, nil, err
	}

	if err = i.setNodeTaints(all, runtimeDriver); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) ScaleDown(mastersToDelete, workersToDelete []net.IP) (registry.Driver, runtime.Driver, error) {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	master0 := masters[0]
	registryDeployHosts := []net.IP{master0}
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

	// reconcile registry node if local registry is ha mode.
	if i.regConfig.LocalRegistry != nil && i.regConfig.LocalRegistry.HaMode {
		registryDeployHosts, err = registry.NewInstaller(masters, i.regConfig.LocalRegistry, i.infraDriver, i.Distributor).Reconcile(netutils.RemoveIPs(masters, mastersToDelete))
		if err != nil {
			return nil, nil, err
		}
	}

	registryConfigurator, err := registry.NewConfigurator(registryDeployHosts, crInfo, i.regConfig, i.infraDriver, i.Distributor)
	if err != nil {
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

	if err = registryConfigurator.UninstallFrom(mastersToDelete, workersToDelete); err != nil {
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
