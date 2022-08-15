// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:13 PM
// @File : configurator
//

package registry

import (
	"fmt"
	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"net"
)

// Configurator, Registry配置器，用于配置本地/远程镜像仓库
type Configurator interface {
	// InitRegistry will
	Reconcile() (Driver, error)

	//Upgrade() (Driver, error)
	//Rollback() (Driver, error)
}

type RegistryConfig struct {
	LocalRegistry    *LocalRegistry
	ExternalRegistry *Registry
}

func NewConfigurator(conf RegistryConfig, containerRuntimeInfo containerruntime.Info) (Configurator, error) {
	if conf.LocalRegistry != nil {
		return &localSingletonConfigurator{
			LocalRegistry:        *conf.LocalRegistry,
			ContainerRuntimeInfo: containerRuntimeInfo,
		}, nil
	}
	if conf.ExternalRegistry != nil {
		return &externalConfigurator{Registry: *conf.ExternalRegistry}, nil
	}

	return nil, fmt.Errorf("")
}

type LocalRegistry struct {
	Registry
	DeployHost net.IP
	DataDir    string   `json:"dataDir,omitempty" yaml:"dataDir,omitempty"`
	Cert       *TLSCert `json:"cert,omitempty" yaml:"cert,omitempty"`
}

type TLSCert struct {
	SubjectAltName *SubjectAltName `json:"subjectAltName,omitempty" yaml:"subjectAltName,omitempty"`
}

type SubjectAltName struct {
	DNSNames []string `json:"dnsNames,omitempty" yaml:"dnsNames,omitempty"`
	IPs      []string `json:"ips,omitempty" yaml:"ips,omitempty"`
}

type Registry struct {
	Domain  string        `json:"domain,omitempty" yaml:"domain,omitempty"`
	Port int           `json:"port,omitempty" yaml:"port,omitempty"`
	Auth *RegistryAuth `json:"auth,omitempty" yaml:"auth,omitempty"`
}

type RegistryAuth struct {
	Username string `json:"username" yaml:"username"`
	Password string `json:"password" yaml:"password"`
}
