// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:29 PM
// @File : installer
//

package container_runtime

import "net"

// 容器运行时安装器
type Installer interface {
	InstallOn([]net.IP) (Info, error)

	UnInstallFrom([]net.IP) error

	//Upgrade() (ContainerRuntimeInfo, error)
	//Rollback() (ContainerRuntimeInfo, error)
}

func NewInstaller(conf Config) Installer {

}

type Config struct {
	Type         string
	LimitNofile  string `json:"limitNofile,omitempty" yaml:"limitNofile,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty" yaml:"cgroupDriver,omitempty"`
}

type Info struct {
	Config
	CRISocket string
}

// 实现
type dockerInstaller struct{}

//type containerdInstaller struct{}
type customContainerRuntimeInstaller struct{}
