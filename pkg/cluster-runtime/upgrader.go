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

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/application"
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/registry"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func (i *Installer) Upgrade() error {
	var (
		masters   = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0   = masters[0]
		workers   = getWorkerIPList(i.infraDriver)
		all       = append(masters, workers...)
		rootfs    = i.infraDriver.GetClusterRootfsPath()
		cmds      = i.infraDriver.GetClusterLaunchCmds()
		appNames  = i.infraDriver.GetClusterLaunchApps()
		extension = i.ImageSpec.ImageExtension
	)

	if extension.Type != imagev1.KubeInstaller {
		return fmt.Errorf("exit upgrade process, wrong cluster image type: %s", extension.Type)
	}

	// distribute rootfs
	if err := i.Distributor.Distribute(all, rootfs); err != nil {
		return err
	}

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

	if err := i.runClusterHook(master0, UpgradeCluster); err != nil {
		return err
	}

	//CMD

	appInstaller := NewAppInstaller(i.infraDriver, i.Distributor, extension)

	v2App, err := application.NewV2Application(v2.ConstructApplication(i.Application, cmds, appNames), extension)
	if err != nil {
		return fmt.Errorf("failed to parse application:%v ", err)
	}

	if err = appInstaller.Launch(master0, v2App.GetImageLaunchCmds()); err != nil {
		return err
	}

	return nil
}
