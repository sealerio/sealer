// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 10:29 PM
// @File : installer
//

package container_runtime

import (
	"fmt"
	"net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/registry"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/platform"
	"github.com/sealerio/sealer/utils/ssh"
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
	InstallOn(hosts []net.IP) (Info, error)

	UnInstallFrom(hosts []net.IP) error

	//Upgrade() (ContainerRuntimeInfo, error)
	//Rollback() (ContainerRuntimeInfo, error)
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

type Config struct {
	Type         string
	LimitNofile  string `json:"limitNofile,omitempty" yaml:"limitNofile,omitempty"`
	CgroupDriver string `json:"cgroupDriver,omitempty" yaml:"cgroupDriver,omitempty"`
}

type Info struct {
	conf      Config
	CRISocket string
}

// 实现
type DockerInstaller struct {
	info    Info
	cluster *v2.Cluster
}

func (d DockerInstaller) InstallOn(hosts []net.IP) (Info, error) {
	rootfs := fmt.Sprintf(common.DefaultTheClusterRootfsDir(d.cluster.Name))
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		ssh, err := ssh.NewStdoutSSHClient(IP, d.cluster)
		if err != nil {
			fmt.Errorf("new ssh client failed: %s", err)
		}
		registryConfig := registry.GetConfig(platform.DefaultMountClusterImageDir(d.cluster.Name), IP)
		initCmd := fmt.Sprintf(RemoteChmod, rootfs, registryConfig.Domain, registryConfig.Port, d.info.conf.CgroupDriver, d.info.conf.LimitNofile)
		err = ssh.CmdAsync(IP, initCmd)
		if err != nil {
			fmt.Errorf("remote exec cmd failed: %s", err)
		}
	}
	return d.info, nil
}

func (d DockerInstaller) UnInstallFrom(hosts []net.IP) error {
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		client, err := ssh.NewStdoutSSHClient(IP, d.cluster)
		if err != nil {
			return fmt.Errorf("new ssh client failed: %s", err)
		}
		err = client.CmdAsync(IP, CleanCmd)
		if err != nil {
			return fmt.Errorf("remote exec clean cmd failed: %s", err)
		}
	}
	return nil
}

//type containerdInstaller struct{}

type ContainerdInstaller struct {
	DockerInstaller
}

func (c ContainerdInstaller) InstallOn(hosts []net.IP) (Info, error) {
	c.info.conf.Type = Containerd
	c.info.CRISocket = DefaultContainerdCRISocket
	rootfs := fmt.Sprintf(common.DefaultTheClusterRootfsDir(c.cluster.Name))
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		client, err := ssh.NewStdoutSSHClient(IP, c.cluster)
		if err != nil {
			fmt.Errorf("new ssh client failed: %s", err)
		}
		registryConfig := registry.GetConfig(platform.DefaultMountClusterImageDir(c.cluster.Name), IP)
		initCmd := fmt.Sprintf(ContainerdRemoteChmod, rootfs, registryConfig.Domain, registryConfig.Port)
		err = client.CmdAsync(IP, initCmd)
		if err != nil {
			fmt.Errorf("remote exec cmd failed: %s", err)
		}
	}
	return c.info, nil
}

func (c ContainerdInstaller) UnInstallFrom(hosts []net.IP) error {
	for ip := range hosts {
		IP := net.ParseIP(string(ip))
		client, err := ssh.NewStdoutSSHClient(IP, c.cluster)
		if err != nil {
			return fmt.Errorf("new ssh client failed: %s", err)
		}
		err = client.CmdAsync(IP, CleanCmd)
		if err != nil {
			return fmt.Errorf("remote exec clean cmd failed: %s", err)
		}
	}
	return nil
}
