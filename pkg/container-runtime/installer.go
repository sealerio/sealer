// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:29 PM
// @File : installer
//

package container_runtime

import (
	"net"
)

const (
	DefaultDockerCRISocket     = "/var/run/dockershim.sock"
	DefaultContainerdCRISocket = "/run/containerd/containerd.sock"
	DefaultSystemdDriver       = "systemd"
	DefaultCgroupfsDriver      = "cgroupfs"
	Docker                     = "docker"
	RemoteChmod                = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh /var/lib/docker %s %s %s %s"
	CleanCmd                   = "cd %s  && chmod +x scripts/* && cd scripts && bash clean.sh"
	ContainerdRemoteChmod      = "cd %s  && chmod +x scripts/* && cd scripts && bash init.sh %s %s"
	DefaultLimitNoFile         = "infinity"
	Containerd                 = "containerd"
)

// 容器运行时安装器
type Installer interface {
	InstallOn(hosts []net.IP) (*Info, error)

	UnInstallFrom(hosts []net.IP) error

	//Upgrade() (ContainerRuntimeInfo, error)
	//Rollback() (ContainerRuntimeInfo, error)
}

type Config struct {
	Type         string
	LimitNofile  string `json:"limitNofile,omitempty" yaml:"limitNofile,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty" yaml:"cgroupDriver,omitempty"`
}

type Info struct {
	conf      Config
	CRISocket string
}

func NewInstaller(conf Config) Installer {
	if conf.Type == "" {
		conf.Type = Docker
	}
	if conf.LimitNofile == "" {
		conf.LimitNofile = DefaultLimitNoFile
	}
	if conf.CgroupDriver == "" {
		conf.CgroupDriver = DefaultSystemdDriver
	}
	info := Info{}
	if info.CRISocket == "" {
		info.CRISocket = DefaultDockerCRISocket
	}
	return DockerInstaller{
		info: Info{
			conf: Config{
				Type:         conf.Type,
				LimitNofile:  conf.LimitNofile,
				CgroupDriver: conf.CgroupDriver,
			},
		},
	}
}
