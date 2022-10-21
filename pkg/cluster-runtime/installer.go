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

	// todo maybe we can support custom registry config later

	clusterENV := infraDriver.GetClusterEnv()

	var registryConfig = registry.Registry{
		Domain: registry.DefaultDomain,
		Port:   registry.DefaultPort,
		Auth:   &registry.Auth{},
	}
	if domain := clusterENV["RegistryDomain"]; domain != nil {
		registryConfig.Domain = domain.(string)
	}
	if userName := clusterENV["RegistryUsername"]; userName != nil {
		registryConfig.Auth.Username = userName.(string)
	}
	if password := clusterENV["RegistryPassword"]; password != nil {
		registryConfig.Auth.Password = password.(string)
	}

	// configure cluster registry
	installer.RegistryConfig.LocalRegistry = &registry.LocalRegistry{
		DataDir:      filepath.Join(infraDriver.GetClusterRootfsPath(), "registry"),
		InsecureMode: false,
		Cert:         &registry.TLSCert{},
		DeployHost:   infraDriver.GetHostIPListByRole(common.MASTER)[0],
		Registry:     registryConfig,
	}

	// add installer hooks
	hooks, err := transferPluginsToHooks(runtimeConfig.Plugins)
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

func (i *Installer) Install() error {
	var (
		masters           = i.infraDriver.GetHostIPListByRole(common.MASTER)
		master0           = masters[0]
		workers           = getWorkerIPList(i.infraDriver)
		all               = append(masters, workers...)
		image             = i.infraDriver.GetClusterImageName()
		clusterLaunchCmds = i.infraDriver.GetClusterLaunchCmds()
	)

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
		err = i.installKubeCluster(master0, all, cmds)
		if err != nil {
			return fmt.Errorf("failed to install cluster: %s", err)
		}
	}

	return nil
}

func (i *Installer) installApp(master0 net.IP, cmds []string) error {
	crInfo, err := i.containerRuntimeInstaller.GetInfo()
	if err != nil {
		return err
	}

	registryConfigurator, err := registry.NewConfigurator(i.RegistryConfig, crInfo, i.infraDriver, i.Distributor)
	if err != nil {
		return err
	}

	registryDriver, err := registryConfigurator.GetDriver()
	if err != nil {
		return err
	}

	err = registryDriver.UploadContainerImages2Registry()
	if err != nil {
		return err
	}

	if err = i.launchClusterImage(master0, cmds); err != nil {
		return err
	}

	return nil
}

func (i *Installer) installKubeCluster(master0 net.IP, all []net.IP, cmds []string) error {
	// distribute rootfs
	if err := i.Distributor.DistributeRootfs(all, i.infraDriver.GetClusterRootfsPath()); err != nil {
		return err
	}

	// set HostAlias
	if err := i.infraDriver.SetClusterHostAliases(all); err != nil {
		return err
	}

	if err := i.runClusterHook(master0, PreInstallCluster); err != nil {
		return err
	}

	if err := i.runHostHook(PreInitHost, all); err != nil {
		return err
	}

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

	if err = i.launchClusterImage(master0, cmds); err != nil {
		return err
	}

	if err = i.runClusterHook(master0, PostInstallCluster); err != nil {
		return err
	}

	if err = i.runHostHook(PostInitHost, all); err != nil {
		return err
	}

	return nil
}

func (i *Installer) launchClusterImage(master0 net.IP, cmds []string) error {
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
