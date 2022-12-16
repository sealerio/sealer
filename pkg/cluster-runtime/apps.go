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
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sealerio/sealer/pkg/rootfs"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	v1 "github.com/sealerio/sealer/pkg/define/application/v1"
	"github.com/sealerio/sealer/pkg/define/application/version"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sirupsen/logrus"
)

type AppInstaller struct {
	infraDriver infradriver.InfraDriver
	distributor imagedistributor.Distributor
	extension   v12.ImageExtension
}

func NewAppInstaller(infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor, extension v12.ImageExtension) AppInstaller {
	return AppInstaller{
		infraDriver: infraDriver,
		distributor: distributor,
		extension:   extension,
	}
}

func (i *AppInstaller) Install(master0 net.IP, cmds []string) error {
	masters := i.infraDriver.GetHostIPListByRole(common.MASTER)
	regConfig := i.infraDriver.GetClusterRegistryConfig()
	// distribute rootfs
	if err := i.distributor.Distribute([]net.IP{master0}, i.infraDriver.GetClusterRootfsPath()); err != nil {
		return err
	}

	//if we use local registry service, load container image to registry
	if regConfig.LocalRegistry != nil {
		deployHosts := masters
		if !regConfig.LocalRegistry.HaMode {
			deployHosts = []net.IP{masters[0]}
		}

		registryConfigurator, err := registry.NewConfigurator(deployHosts, containerruntime.Info{}, regConfig, i.infraDriver, i.distributor)
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
	}

	if err := i.LaunchClusterImage(master0, cmds); err != nil {
		return err
	}

	return nil
}

func (i AppInstaller) LaunchClusterImage(master0 net.IP, launchCmds []string) error {
	var (
		cmds       []string
		rootfsPath = i.infraDriver.GetClusterRootfsPath()
		ex         = shell.NewLex('\\')
	)

	if len(launchCmds) > 0 {
		cmds = launchCmds
	} else {
		cmds = GetImageDefaultLaunchCmds(i.extension)
	}

	for _, value := range cmds {
		if value == "" {
			continue
		}
		cmdline, err := ex.ProcessWordWithMap(value, map[string]string{})
		if err != nil {
			return fmt.Errorf("failed to render launch cmd: %v", err)
		}

		if err = i.infraDriver.CmdAsync(master0, fmt.Sprintf(common.CdAndExecCmd, rootfsPath, cmdline)); err != nil {
			return err
		}
	}

	return i.save(common.GetDefaultApplicationFile())
}

// todo save image info to disk or api server, we need new interface to do this.
func (i AppInstaller) save(applicationFile string) error {
	f, err := os.OpenFile(filepath.Clean(applicationFile), os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return fmt.Errorf("cannot flock file %s - %s", applicationFile, err)
	}
	defer func() {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		if err != nil {
			logrus.Errorf("failed to unlock %s", applicationFile)
		}
	}()

	content, err := json.Marshal(i.extension)
	if err != nil {
		return fmt.Errorf("failed to marshal image extension: %v", err)
	}

	if _, err = f.Write(content); err != nil {
		return err
	}

	return nil
}

func GetImageDefaultLaunchCmds(extension v12.ImageExtension) []string {
	appNames := extension.Launch.AppNames
	launchCmds := GetAppLaunchCmdsByNames(appNames, extension.Applications)

	// if app name exist in extension, return it`s launch cmds firstly.
	if len(launchCmds) != 0 {
		return launchCmds
	}

	return extension.Launch.Cmds
}

func GetAppLaunchCmdsByNames(appNames []string, apps []version.VersionedApplication) []string {
	var appCmds []string
	for _, name := range appNames {
		appRoot := makeItDir(filepath.Join(rootfs.GlobalManager.App().Root(), name))
		for _, app := range apps {
			v1app := app.(*v1.Application)
			if v1app.Name() != name {
				continue
			}
			appCmds = append(appCmds, v1app.LaunchCmd(appRoot))
		}
	}
	return appCmds
}

func makeItDir(str string) string {
	if !strings.HasSuffix(str, "/") {
		return str + "/"
	}
	return str
}
