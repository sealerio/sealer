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

package registry

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/containers/common/pkg/auth"
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

type externalConfigurator struct {
	containerRuntimeInfo containerruntime.Info
	infraDriver          infradriver.InfraDriver
	endpoint             string
	username             string
	password             string
}

func (e *externalConfigurator) GetRegistryInfo() RegistryInfo {
	clusterRegistry := e.infraDriver.GetClusterRegistry()
	return RegistryInfo{
		External: clusterRegistry.ExternalRegistry,
	}
}

func (e *externalConfigurator) InstallOn(masters, nodes []net.IP) error {
	hosts := append(masters, nodes...)
	var (
		username        = e.username
		password        = e.password
		endpoint        = e.endpoint
		tmpAuthFilePath = "/tmp/config.json"
		// todo we need this config file when kubelet pull images from registry. while, we could optimize the logic here.
		remoteKubeletAuthFilePath = "/var/lib/kubelet/config.json"
	)

	if username == "" || password == "" {
		return nil
	}

	err := auth.Login(context.TODO(),
		nil,
		&auth.LoginOptions{
			AuthFile:           tmpAuthFilePath,
			Password:           password,
			Username:           username,
			Stdout:             os.Stdout,
			AcceptRepositories: true,
		},
		[]string{endpoint})

	if err != nil {
		return err
	}

	defer func() {
		err = os.Remove(tmpAuthFilePath)
		if err != nil {
			logrus.Debugf("failed to remove tmp registry auth file:%s", tmpAuthFilePath)
		}
	}()

	err = e.copy2RemoteHosts(tmpAuthFilePath, e.containerRuntimeInfo.ConfigFilePath, hosts)
	if err != nil {
		return err
	}

	err = e.copy2RemoteHosts(tmpAuthFilePath, remoteKubeletAuthFilePath, hosts)
	if err != nil {
		return err
	}

	return nil
}

func (e *externalConfigurator) copy2RemoteHosts(src, dest string, hosts []net.IP) error {
	f := func(host net.IP) error {
		err := e.infraDriver.Copy(host, src, dest)
		if err != nil {
			return fmt.Errorf("failed to copy local file %s to remote %s : %v", src, dest, err)
		}
		return nil
	}

	return e.infraDriver.Execute(hosts, f)
}

func (e *externalConfigurator) UninstallFrom(masters, nodes []net.IP) error {
	if e.username == "" || e.password == "" {
		return nil
	}
	hosts := append(masters, nodes...)
	//todo use sdk to logout instead of shell cmd
	logoutCmd := fmt.Sprintf("docker logout %s ", e.endpoint)
	//nolint
	if e.containerRuntimeInfo.Type != common.Docker {
		logoutCmd = fmt.Sprintf("nerdctl logout %s ", e.endpoint)
	}

	for _, host := range hosts {
		err := e.infraDriver.CmdAsync(host, nil, logoutCmd)
		if err != nil {
			return fmt.Errorf("failed to delete registry configuration: %v", err)
		}
	}

	return nil
}

func (e *externalConfigurator) GetDriver() (Driver, error) {
	return newExternalRegistryDriver(e.endpoint), nil
}

func NewExternalConfigurator(regConfig *v2.ExternalRegistry, containerRuntimeInfo containerruntime.Info, driver infradriver.InfraDriver) (Configurator, error) {
	domain := regConfig.Domain
	if regConfig.Port != 0 {
		domain = net.JoinHostPort(regConfig.Domain, strconv.Itoa(regConfig.Port))
	}
	return &externalConfigurator{
		endpoint:             domain,
		username:             regConfig.Username,
		password:             regConfig.Password,
		infraDriver:          driver,
		containerRuntimeInfo: containerRuntimeInfo,
	}, nil
}
