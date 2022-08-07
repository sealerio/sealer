// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 5:15 PM
// @File : infradriver
//

package infradriver

import (
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"
	"net"
)

// 基础设施驱动器，将整个集群视作一个操作系统内核，此处的接口对标系统调用
type InfraDriver interface {
	// Copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host net.IP, srcFilePath, dstFilePath string) error
	// CopyR copy remote host files to localhost
	CopyR(host net.IP, srcFilePath, dstFilePath string) error

	// CmdAsync exec command on remote host, and asynchronous return logs
	CmdAsync(host net.IP, cmd ...string) error
	// Cmd exec command on remote host, and return combined standard output and standard error
	Cmd(host net.IP, cmd string) ([]byte, error)
	// CmdToString exec command on remote host, and return spilt standard output and standard error
	CmdToString(host net.IP, cmd, spilt string) (string, error)

	// IsFileExist check remote file exist or not
	IsFileExist(host net.IP, remoteFilePath string) (bool, error)
	// RemoteDirExist Remote file existence returns true, nil
	RemoteDirExist(host net.IP, remoteDirpath string) (bool, error)

	// GetPlatform Get remote platform
	GetPlatform(host net.IP) (v1.Platform, error)
	// Ping Ping remote host
	Ping(host net.IP) error
	// SetHostName add or update host name on host
	SetHostName(host net.IP, hostName string) error
	// SetLvsRule add or update host name on host
	//SetLvsRule(host net.IP, hostName string) error
}

type SSHInfraDriver struct {
	sshConfigs map[string]ssh.Interface
}

func NewInfraDriver(cluster *v2.Cluster) InfraDriver {
	ret := SSHInfraDriver{}

	// init ssh configs for all host, using cluster configuration

	return &ret
}
