// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 9:59 PM
// @File : cluster_runtime
//

package cluster_runtime

import (
	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm_config"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"net"
)

// 初始化配置
type RuntimeConfig struct {
	RegistryConfig         registry.RegistryConfig
	ContainerRuntimeConfig containerruntime.Config
	KubeadmConfig          kubeadm_config.KubeadmConfig
	HookConfig             HookConfig
}

type Installer struct {
	RuntimeConfig
	infraDriver               infradriver.InfraDriver
	containerRuntimeInstaller containerruntime.Installer
	hooks                     map[Phase][]HookFunc
}

func NewInstaller(infra infradriver.InfraDriver, cluster *v2.Cluster) (*Installer, error) {
	//TODO: use cluster set RuntimeConfig and check valid

	installer := Installer{}

	var err error
	installer.containerRuntimeInstaller, err = containerruntime.NewInstaller(conf.ContainerRuntimeConfig, infra)
	if err != nil {
		return nil, err
	}

	installer.RegistryConfig.LocalRegistry.DeployHost = masters[0]

	//TODO: use cluster set hooks
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

func (i *Installer) Install() (registry.Driver, runtime.Driver, error) {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)
	if err := i.runHook(PreInstall); err != nil {
		return nil, nil, err
	}

	if err := i.runHookOnHosts(PreInitHost, all); err != nil {
		return nil, nil, err
	}

	if err := i.containerRuntimeInstaller.InstallOn(all); err != nil {
		return nil, nil, err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	// TODO, init here or in constructor?
	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
	if err != nil {
		return nil, nil, err
	}

	if err := registryConfigurator.Reconcile(); err != nil {
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

	if err := kubeRuntimeInstaller.Install(); err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if err := i.runHookOnHosts(PostInitHost, all); err != nil {
		return nil, nil, err
	}

	if err := i.runHook(PostInstall); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) UnInstall() error {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)

	if err := i.runHook(PreUnInstall); err != nil {
		return err
	}

	if err := i.runHookOnHosts(PreCleanHost, all); err != nil {
		return err
	}

	kubeRuntimeInstaller, err := kubernetes.NewKubeadmRuntime(i.KubeadmConfig, i.infraDriver, containerruntime.Info{}, registry.Info{})
	if err != nil {
		return err
	}

	if err := kubeRuntimeInstaller.Reset(); err != nil {
		return err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, containerruntime.Info{}, i.infraDriver)
	if err != nil {
		return err
	}

	if err := registryConfigurator.Clean(); err != nil {
		return err
	}

	if err := i.containerRuntimeInstaller.UnInstallFrom(all); err != nil {
		return err
	}

	if err := i.runHookOnHosts(PostCleanHost, all); err != nil {
		return err
	}

	if err := i.runHook(PostUnInstall); err != nil {
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
	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
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
	if err := i.runHook(PreScaleUp); err != nil {
		return nil, nil, err
	}

	if err := i.runHookOnHosts(PreInitHost, append(newMasters, newWorkers...)); err != nil {
		return nil, nil, err
	}

	if err := i.containerRuntimeInstaller.InstallOn(append(newMasters, newWorkers...)); err != nil {
		return nil, nil, err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
	if err != nil {
		return nil, nil, err
	}

	if err := registryConfigurator.Reconcile(); err != nil {
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

	if err := i.runHookOnHosts(PostInitHost, append(newMasters, newWorkers...)); err != nil {
		return nil, nil, err
	}

	if err := i.runHook(PostScaleUp); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}

func (i *Installer) ScaleDown(mastersToDelete, workersToDelete []net.IP) (registry.Driver, runtime.Driver, error) {
	if err := i.runHookOnHosts(PreCleanHost, append(mastersToDelete, workersToDelete...)); err != nil {
		return nil, nil, err
	}

	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return nil, nil, err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
	if err != nil {
		return nil, nil, err
	}

	if err := registryConfigurator.Reconcile(); err != nil {
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

	if err := kubeRuntimeInstaller.ScaleDown(mastersToDelete, workersToDelete); err != nil {
		return nil, nil, err
	}

	runtimeDriver, err := kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, nil, err
	}

	if err := i.containerRuntimeInstaller.UnInstallFrom(append(mastersToDelete, workersToDelete...)); err != nil {
		return nil, nil, err
	}

	if err := i.runHookOnHosts(PostCleanHost, append(mastersToDelete, workersToDelete...)); err != nil {
		return nil, nil, err
	}

	return registryDriver, runtimeDriver, nil
}
