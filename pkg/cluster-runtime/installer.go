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
	"github.com/sealerio/sealer/pkg/runtime/kubernetes/kubeadm"
	"net"
)

// 初始化配置
type RuntimeConfig struct {
	RegistryConfig         registry.RegistryConfig
	ContainerRuntimeConfig containerruntime.Config
	KubeadmConfig          *kubeadm.KubeadmConfig
	HookConfig             HookConfig
}

type Installer struct {
	RuntimeConfig
	infraDriver               infradriver.InfraDriver
	registryConfigurator      registry.Configurator
	containerRuntimeInstaller containerruntime.Installer
	kubeRuntimeInstaller      runtime.Interface
	hooks                     map[Phase][]HookFunc
}

func NewInstaller(infra infradriver.InfraDriver, conf RuntimeConfig) (*Installer, error) {
	// set RuntimeConfig and check valid

	// add hooks
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

func (i *Installer) Install() (*Driver, error) {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	workers := getWorkerIPList(i.infraDriver)
	all := append(masters, workers...)
	if err := i.runHook(PreInstall); err != nil {
		return nil, err
	}

	if err := i.runHookOnHosts(PreInitHost, all); err != nil {
		return nil, err
	}

	var err error
	i.containerRuntimeInstaller = containerruntime.NewInstaller(i.ContainerRuntimeConfig)

	crInfo, err := i.containerRuntimeInstaller.InstallOn(all)
	if err != nil {
		return nil, err
	}

	// TODO, how to
	i.RegistryConfig.LocalRegistry.DeployHost = masters[0]

	// TODO, init here or in constructor?
	i.registryConfigurator, err = registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
	if err != nil {
		return nil, err
	}

	registryDriver, err := i.registryConfigurator.Reconcile()
	if err != nil {
		return nil, err
	}

	i.kubeRuntimeInstaller, err = kubernetes.NewDefaultRuntime(i.KubeadmConfig, registryDriver.GetInfo(), crInfo)
	if err != nil {
		return nil, err
	}

	if err := i.kubeRuntimeInstaller.Init(); err != nil {
		return nil, err
	}

	runtimeDriver, err := i.kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, err
	}

	if err := i.runHookOnHosts(PostInitHost, all); err != nil {
		return nil, err
	}

	if err := i.runHook(PostInstall); err != nil {
		return nil, err
	}

	return &Driver{
		RegistryDriver:    registryDriver,
		KubeRuntimeDriver: runtimeDriver,
	}, nil
}

func (i *Installer) UnInstall() error {}

func (i *Installer) GetCurrentDriver() (*Driver, error) {}

func (i *Installer) ScaleUp(newMasters, newWorkers []net.IP) (*Driver, error) {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)

	if err := i.runHook(PreInstall); err != nil {
		return nil, err
	}

	if err := i.runHookOnHosts(PreInitHost, append(newMasters, newWorkers...)); err != nil {
		return nil, err
	}

	var err error
	i.containerRuntimeInstaller = containerruntime.NewInstaller(i.ContainerRuntimeConfig)

	crInfo, err := i.containerRuntimeInstaller.InstallOn(append(newMasters, newWorkers...))
	if err != nil {
		return nil, err
	}

	// TODO, how to
	i.RegistryConfig.LocalRegistry.DeployHost = masters[0]

	i.registryConfigurator, err = registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver)
	if err != nil {
		return nil, err
	}

	registryDriver, err := i.registryConfigurator.Reconcile()
	if err != nil {
		return nil, err
	}

	i.kubeRuntimeInstaller, err = kubernetes.NewDefaultRuntime(i.KubeadmConfig, registryDriver.GetInfo(), crInfo)
	if err != nil {
		return nil, err
	}

	if err := i.kubeRuntimeInstaller.JoinMasters(newMasters); err != nil {
		return nil, err
	}

	if err := i.kubeRuntimeInstaller.JoinNodes(newWorkers); err != nil {
		return nil, err
	}

	runtimeDriver, err := i.kubeRuntimeInstaller.GetCurrentRuntimeDriver()
	if err != nil {
		return nil, err
	}

	if err := i.runHookOnHosts(PostInitHost, append(newMasters, newWorkers...)); err != nil {
		return nil, err
	}

	if err := i.runHook(PostInstall); err != nil {
		return nil, err
	}

	return &Driver{
		RegistryDriver:    registryDriver,
		KubeRuntimeDriver: runtimeDriver,
	}, nil
}

func (i *Installer) ScaleDown(hosts []net.IP) (*Driver, error) {

}
