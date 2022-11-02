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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"path/filepath"

	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	v12 "github.com/sealerio/sealer/pkg/define/image/v1"
	"github.com/sealerio/sealer/pkg/imagedistributor"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	osutils "github.com/sealerio/sealer/utils/os"
)

type AppInstaller struct {
	infraDriver    infradriver.InfraDriver
	distributor    imagedistributor.Distributor
	registryConfig registry.RegConfig
	extension      v12.ImageExtension
}

func NewAppInstaller(infraDriver infradriver.InfraDriver, distributor imagedistributor.Distributor, extension v12.ImageExtension, registryConfig registry.RegConfig) AppInstaller {
	return AppInstaller{
		infraDriver:    infraDriver,
		distributor:    distributor,
		registryConfig: registryConfig,
		extension:      extension,
	}
}

func (i *AppInstaller) Install(master0 net.IP, cmds []string) error {
	// distribute rootfs
	if err := i.distributor.DistributeRootfs([]net.IP{master0}, i.infraDriver.GetClusterRootfsPath()); err != nil {
		return err
	}

	registryConfigurator, err := registry.NewConfigurator(i.registryConfig, containerruntime.Info{}, i.infraDriver, i.distributor)
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

	if err = i.Launch(master0, cmds); err != nil {
		return err
	}

	return i.save()
}

func (i AppInstaller) Launch(master0 net.IP, launchCmds []string) error {
	var (
		cmds          []string
		clusterRootfs = i.infraDriver.GetClusterRootfsPath()
		ex            = shell.NewLex('\\')
	)

	if len(launchCmds) > 0 {
		cmds = launchCmds
	} else {
		cmds = i.extension.Launch.Cmds
	}

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

// todo save image info to disk or api server, we need new interface to do this.
func (i AppInstaller) save() error {
	var extensionList []v12.ImageExtension
	applicationFile := common.GetDefaultApplicationFile()

	if osutils.IsFileExist(applicationFile) {
		b, err := ioutil.ReadFile(filepath.Clean(applicationFile))
		if err != nil {
			return err
		}

		b = bytes.TrimSpace(b)
		if len(b) != 0 {
			if err := json.Unmarshal(b, &extensionList); err != nil {
				return fmt.Errorf("failed to load default application file %s: %v", applicationFile, err)
			}
		}
	}

	extensionList = append(extensionList, i.extension)
	content, err := json.Marshal(extensionList)
	if err != nil {
		return fmt.Errorf("failed to marshal image extension: %v", err)
	}

	return osutils.NewCommonWriter(applicationFile).WriteFile(content)
}
