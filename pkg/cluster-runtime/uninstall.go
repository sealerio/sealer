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
	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
)

func (i *Installer) UnInstall() error {
	master0 := i.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)
	// delete HostAlias
	if err := i.infraDriver.DeleteClusterHostAliases(all); err != nil {
		return err
	}

	if err := i.runClusterHook(master0, PreUnInstallCluster); err != nil {
		return err
	}

	if err := i.runHostHook(PreCleanHost, all); err != nil {
		return err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, containerruntime.Info{}, registry.Info{})
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

	registryConfigurator, err := registry.NewConfigurator(crInfo, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	if err = registryConfigurator.UninstallFrom(all); err != nil {
		return err
	}

	if err = registryConfigurator.Clean(); err != nil {
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

	if err = i.Distributor.Restore(i.infraDriver.GetClusterBasePath(), all); err != nil {
		return err
	}

	return nil
}
