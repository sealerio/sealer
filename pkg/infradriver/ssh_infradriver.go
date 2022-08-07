// alibaba-inc.com Inc.
// Copyright (c) 2004-2022 All Rights Reserved.
//
// @Author : huaiyou.cyz
// @Time : 2022/8/7 5:44 PM
// @File : ssh_infradriver
//

package infradriver

import (
	v1 "github.com/sealerio/sealer/types/api/v1"
	"net"
)

func (d *SSHInfraDriver) Copy(host net.IP, srcFilePath, dstFilePath string) error {

}

func (d *SSHInfraDriver) CopyR(host net.IP, srcFilePath, dstFilePath string) error {

}

func (d *SSHInfraDriver) CmdAsync(host net.IP, cmd ...string) error {

}

func (d *SSHInfraDriver) Cmd(host net.IP, cmd string) ([]byte, error) {

}

func (d *SSHInfraDriver) CmdToString(host net.IP, cmd, spilt string) (string, error) {

}

func (d *SSHInfraDriver) IsFileExist(host net.IP, remoteFilePath string) (bool, error) {

}

func (d *SSHInfraDriver) RemoteDirExist(host net.IP, remoteDirpath string) (bool, error) {

}

func (d *SSHInfraDriver) GetPlatform(host net.IP) (v1.Platform, error) {

}

func (d *SSHInfraDriver) Ping(host net.IP) error {

}

// CopyR copy remote host files to localhost

// CmdAsync exec command on remote host, and asynchronous return logs

// Cmd exec command on remote host, and return combined standard output and standard error

// CmdToString exec command on remote host, and return spilt standard output and standard error

// IsFileExist check remote file exist or not

// RemoteDirExist Remote file existence returns true, nil

// GetPlatform Get remote platform

// Ping Ping remote host
