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
	"path/filepath"

	"github.com/moby/buildkit/frontend/dockerfile/shell"

	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	common2 "github.com/sealerio/sealer/pkg/define/options"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/imageengine"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	v1 "github.com/sealerio/sealer/types/api/v1"
)

var ForceDelete bool

// RuntimeConfig for Installer
type RuntimeConfig struct {
	ImageEngine            imageengine.Interface
	Distributor            imagedistributor.Distributor
	RegistryConfig         registry.RegConfig
	ContainerRuntimeConfig containerruntime.Config
	KubeadmConfig          kubeadm.KubeadmConfig
	Plugins                []v1.Plugin
}

type Installer struct {
	RuntimeConfig
	infraDriver               infradriver.InfraDriver
	containerRuntimeInstaller containerruntime.Installer
	hooks                     map[Phase]HookConfigList
}

func NewInstaller(infraDriver infradriver.InfraDriver, runtimeConfig RuntimeConfig) (*Installer, error) {
	var (
		err       error
		installer = &Installer{}
	)

	// configure container runtime
	//todo need to support other container runtimes
	installer.containerRuntimeInstaller, err = containerruntime.NewInstaller(containerruntime.Config{
		Type:         "docker",
		CgroupDriver: "systemd",
	}, infraDriver)
	if err != nil {
		return nil, err
	}

	// configure cluster registry
	installer.RegistryConfig.LocalRegistry = &registry.LocalRegistry{
		DataDir:      filepath.Join(infraDriver.GetClusterRootfsPath(), "registry"),
		InsecureMode: false,
		Cert:         &registry.TLSCert{},
		DeployHost:   infraDriver.GetHostIPListByRole(common.MASTER)[0],
		Registry: registry.Registry{
			Domain: registry.DefaultDomain,
			Port:   registry.DefaultPort,
			Auth:   &registry.Auth{},
		},
	}

	// add installer hooks
	plugins, err := LoadPluginsFromFile(filepath.Join(infraDriver.GetClusterRootfsPath(), "plugins"))
	if err != nil {
		return nil, err
	}
	plugins = append(plugins, runtimeConfig.Plugins...)
	hooks, err := transferPluginsToHooks(plugins)
	if err != nil {
		return nil, err
	}

	installer.hooks = hooks
	installer.infraDriver = infraDriver
	installer.KubeadmConfig = runtimeConfig.KubeadmConfig
	installer.Distributor = runtimeConfig.Distributor
	installer.ImageEngine = runtimeConfig.ImageEngine

	return installer, nil
}

func getWorkerIPList(infraDriver infradriver.InfraDriver) []net.IP {
	masters := make(map[string]bool)
	for _, master := range infraDriver.GetHostIPListByRole(common.MASTER) {
		masters[master.String()] = true
	}
	all := infraDriver.GetHostIPList()
	workers := make([]net.IP, len(all)-len(masters))

	index := 0
	for _, ip := range all {
		if !masters[ip.String()] {
			workers[index] = ip
			index++
		}
	}

	return workers
}

func (i *Installer) Install() error {
	var (
		masters           = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0           = masters[0]
		workers           = getWorkerIPList(i.infraDriver)
		all               = append(masters, workers...)
		image             = i.infraDriver.GetClusterImageName()
		clusterLaunchCmds = i.infraDriver.GetClusterLaunchCmds()
	)

	// distribute rootfs
	if err := i.Distributor.DistributeRootfs(all, i.infraDriver.GetClusterRootfsPath()); err != nil {
		return err
	}

	if err := i.runClusterHook(master0, PreInstallCluster); err != nil {
		return err
	}

	if err := i.runHostHook(PreInitHost, all); err != nil {
		return err
	}

	extension, err := i.ImageEngine.GetSealerImageExtension(&common2.GetImageAnnoOptions{ImageNameOrID: image})
	if err != nil {
		return fmt.Errorf("failed to get ClusterImage extension: %s", err)
	}

	var cmds []string
	if len(clusterLaunchCmds) > 0 {
		cmds = clusterLaunchCmds
	} else {
		cmds = extension.Launch.Cmds
	}

	if extension.Type == v12.AppInstaller {
		err = i.installApp(master0, cmds)
		if err != nil {
			return fmt.Errorf("failed to install application: %s", err)
		}
	} else {
		err = i.installKubeCluster(all)
		if err != nil {
			return fmt.Errorf("failed to install cluster: %s", err)
		}

		err = i.installApp(master0, cmds)
		if err != nil {
			return fmt.Errorf("failed to install application: %s", err)
		}
	}

	if err = i.runClusterHook(master0, PostInstallCluster); err != nil {
		return err
	}

	if err = i.runHostHook(PostInitHost, all); err != nil {
		return err
	}

	return nil
}

func (i *Installer) installKubeCluster(all []net.IP) error {
	if err := i.containerRuntimeInstaller.InstallOn(all); err != nil {
		return err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	if err = registryConfigurator.Reconcile(all); err != nil {
		return err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, crInfo, registryDriver.GetInfo())
	if err != nil {
		return err
	}

	if err = kubeRuntimeInstaller.Install(); err != nil {
		return err
	}

	return nil
}

func (i *Installer) installApp(master0 net.IP, cmds []string) error {
	var (
		clusterRootfs = i.infraDriver.GetClusterRootfsPath()
		ex            = shell.NewLex('\\')
	)

	for _, value := range cmds {
		if value == "" {
			continue
		}
		cmdline, err := ex.ProcessWordWithMap(value, map[string]string{})
		if err != nil {
			return fmt.Errorf("failed to render launch cmd: %v", err)
		}

		if err = i.infraDriver.CmdAsync(master0, fmt.Sprintf(common.CdAndExecCmd, clusterRootfs, cmdline)); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) UnInstall() error {
	master0 := i.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)

	if err := confirmDeleteHosts(fmt.Sprintf("%s/%s", common.MASTER, common.NODE), all); err != nil {
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

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
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

func (i *Installer) GetCurrentDriver() (registry.Driver, runtime.Driver, error) {
	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	// TODO, init here or in constructor?
	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
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

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) ScaleUp(newMasters, newWorkers []net.IP) (registry.Driver, runtime.Driver, error) {
	master0 := i.infraDriver.GetHostIPListByRole(common.MASTER)[0]
	all := append(newMasters, newWorkers...)

	// distribute rootfs
	if err := i.Distributor.DistributeRootfs(all, i.infraDriver.GetClusterRootfsPath()); err != nil {
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

	if err := registryConfigurator.Reconcile(all); err != nil {
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
