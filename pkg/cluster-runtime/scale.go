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
	"fmt"
	"net"

	netutils "github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sirupsen/logrus"
)

func (i *Installer) ScaleUp(newMasters, newWorkers []net.IP) (registry.Driver, runtime.Driver, error) {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	master0 := masters[0]
	workers := getWorkerIPList(i.infraDriver)
	registryDeployHosts := []net.IP{master0}
	all := append(newMasters, newWorkers...)
	rootfs := i.infraDriver.GetClusterRootfsPath()

	logrus.Debug("check ssh of new nodes")
	_, err := CheckNodeSSH(i.infraDriver, append(newMasters, newWorkers...))
	if err != nil {
		return nil, nil, err
	}

	if len(newMasters) != 0 {
		logrus.Debug("check ssh of workers")
		_, err = CheckNodeSSH(i.infraDriver, workers)
		if err != nil {
			return nil, nil, err
		}
	}

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
	if i.regConfig.LocalRegistry != nil && *i.regConfig.LocalRegistry.HA {
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

	kubeRuntimeInstaller, err := getClusterRuntimeInstaller(i.clusterRuntimeType, i.infraDriver,
		crInfo, registryDriver.GetInfo(), i.KubeadmConfig)
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

	if err := i.setRoles(runtimeDriver); err != nil {
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
	workers := getWorkerIPList(i.infraDriver)
	remainWorkers := netutils.RemoveIPs(workers, workersToDelete)

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	registryDeployHosts := []net.IP{master0}
	// reconcile registry node if local registry is ha mode.
	if i.regConfig.LocalRegistry != nil && *i.regConfig.LocalRegistry.HA {
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

	kubeRuntimeInstaller, err := getClusterRuntimeInstaller(i.clusterRuntimeType, i.infraDriver,
		crInfo, registryDriver.GetInfo(), i.KubeadmConfig)
	if err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if len(mastersToDelete) != 0 {
		logrus.Debug("check ssh of remainWorkers")
		_, err := CheckNodeSSH(i.infraDriver, remainWorkers)
		if err != nil {
			return nil, nil, fmt.Errorf("because master list changed, we need connect to all existing workers to maintain some configs, but failed: %v", err)
		}
	}

	logrus.Debug("check ssh of nodesToDelete")
	disconnetedMasters, err := CheckNodeSSH(i.infraDriver, mastersToDelete)
	if err != nil {
		logrus.Warn(err.Error())
	}
	disconnetedWorkers, err := CheckNodeSSH(i.infraDriver, workersToDelete)
	if err != nil {
		logrus.Warn(err.Error())
	}

	if err := i.resetAndScaleDown(kubeRuntimeInstaller, registryConfigurator, netutils.RemoveIPs(mastersToDelete, disconnetedMasters), netutils.RemoveIPs(workersToDelete, disconnetedWorkers)); err != nil {
		return nil, nil, err
	}

	if err := i.onlyScaleDown(kubeRuntimeInstaller, disconnetedMasters, disconnetedWorkers); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) resetAndScaleDown(kubeRuntimeInstaller runtime.Installer, registryConfigurator registry.Configurator, mastersToDelete, workersToDelete []net.IP) error {
	allToDelete := append(mastersToDelete, workersToDelete...)

	if err := i.runHostHook(PreCleanHost, allToDelete); err != nil {
		return err
	}

	if err := kubeRuntimeInstaller.ScaleDown(mastersToDelete, workersToDelete); err != nil {
		return err
	}

	if err := registryConfigurator.UninstallFrom(mastersToDelete, workersToDelete); err != nil {
		return err
	}

	if err := i.containerRuntimeInstaller.UnInstallFrom(allToDelete); err != nil {
		return err
	}

	if err := i.runHostHook(PostCleanHost, allToDelete); err != nil {
		return err
	}

	// delete HostAlias
	if err := i.infraDriver.DeleteClusterHostAliases(allToDelete); err != nil {
		return err
	}

	if err := i.Distributor.Restore(i.infraDriver.GetClusterBasePath(), allToDelete); err != nil {
		return err
	}

	return nil
}

func (i *Installer) onlyScaleDown(kubeRuntimeInstaller runtime.Installer, mastersToDelete, workersToDelete []net.IP) error {
	if err := kubeRuntimeInstaller.ScaleDown(mastersToDelete, workersToDelete); err != nil {
		return err
	}

	return nil
}
