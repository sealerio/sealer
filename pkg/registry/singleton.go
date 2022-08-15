// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:19 PM
// @File : local
//

package registry

import (
	"fmt"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"path/filepath"
)

const (
	RegistryName                = "sealer-registry"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DockerLoginCommand          = "nerdctl login -u %s -p %s %s && " + KubeletAuthCommand
	KubeletAuthCommand          = "cp /root/.docker/config.json /var/lib/kubelet"
	DeleteRegistryCommand       = "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
)

type localSingletonConfigurator struct {
	rootfs string
	LocalRegistry
	infraDriver          infradriver.InfraDriver
	ContainerRuntimeInfo containerruntime.Info
}

// Reconcile local private registry by rootfs scripts.
func (c *localSingletonConfigurator) Reconcile() (Driver, error) {
	// 1. gen tls cert: gen by default, if exist will skip.
	// 2. gen auth info: if not config,skip, if auth file exist,will skip
	// 3. bash init-registry.sh ${port} ${mountData} ${domain}

	if err := c.genTLSCerts(); err != nil {
		return nil, err
	}

	if err := c.genBasicAuth(); err != nil {
		return nil, err
	}

	if err := c.reconcile(); err != nil {
		return nil, err
	}

}

func (c *localSingletonConfigurator) genTLSCerts() error {
	// 1. gen tls cert: gen by default
}

func (c *localSingletonConfigurator) genBasicAuth() error {
	//gen basic auth info: if not config, will skip.
	if c.Auth.Username == "" || c.Auth.Password == "" {
		return nil
	}

	var basicAuthFile = filepath.Join(c.rootfs, "etc", DefaultRegistryHtPasswdFile)

	htpasswd, err := GenerateHTTPBasicAuth(c.Auth.Username, c.Auth.Password)
	if err != nil {
		return err
	}

	writeCMD := fmt.Sprintf("echo '%s' > %s", htpasswd, basicAuthFile)

	err = c.infraDriver.CmdAsync(c.DeployHost, writeCMD)
	if err != nil {
		return err
	}

	return nil
}

func (c *localSingletonConfigurator) reconcile() error {
	// 3. bash init-registry.sh ${port} ${mountData} ${domain}
	initRegistry := fmt.Sprintf("cd %s/scripts && bash init-registry.sh %s %s %s", c.rootfs, c.Port, c.DataDir, c.Domain)
	if err := c.infraDriver.CmdAsync(c.DeployHost, initRegistry); err != nil {
		return err
	}

	return nil
}
