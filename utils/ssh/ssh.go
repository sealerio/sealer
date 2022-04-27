// Copyright © 2021 Alibaba Group Holding Ltd.
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

package ssh

import (
	"fmt"
	"net"
	"time"

	"github.com/imdario/mergo"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
)

type Interface interface {
	// Copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host, srcFilePath, dstFilePath string) error
	// Fetch copy remote host files to localhost
	Fetch(host, srcFilePath, dstFilePath string) error
	// CmdAsync exec command on remote host, and asynchronous return logs
	CmdAsync(host string, cmd ...string) error
	// Cmd exec command on remote host, and return combined standard output and standard error
	Cmd(host, cmd string) ([]byte, error)
	// IsFileExist check remote file exist or not
	IsFileExist(host, remoteFilePath string) (bool, error)
	// RemoteDirExist Remote file existence returns true, nil
	RemoteDirExist(host, remoteDirpath string) (bool, error)
	// CmdToString exec command on remote host, and return spilt standard output and standard error
	CmdToString(host, cmd, spilt string) (string, error)
	// Platform Get remote platform
	Platform(host string) (v1.Platform, error)

	Ping(host string) error
}

type SSH struct {
	IsStdout     bool
	Encrypted    bool
	User         string
	Password     string
	Port         string
	PkFile       string
	PkPassword   string
	Timeout      *time.Duration
	LocalAddress []net.Addr
}

func NewSSHClient(ssh *v1.SSH, isStdout bool) Interface {
	if ssh.User == "" {
		ssh.User = common.ROOT
	}
	address, err := utils.GetLocalHostAddresses()
	if err != nil {
		logger.Warn("failed to get local address, %v", err)
	}
	return &SSH{
		IsStdout:     isStdout,
		Encrypted:    ssh.Encrypted,
		User:         ssh.User,
		Password:     ssh.Passwd,
		Port:         ssh.Port,
		PkFile:       ssh.Pk,
		PkPassword:   ssh.PkPasswd,
		LocalAddress: address,
	}
}

// GetHostSSHClient is used to executed bash command and no std out to be printed.
func GetHostSSHClient(hostIP string, cluster *v2.Cluster) (Interface, error) {
	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			if hostIP == ip {
				if err := mergo.Merge(&host.SSH, &cluster.Spec.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&host.SSH, false), nil
			}
		}
	}
	return nil, fmt.Errorf("get host ssh client failed, host ip %s not in hosts ip list", hostIP)
}

// NewStdoutSSHClient is used to show std out when execute bash command.
func NewStdoutSSHClient(hostIP string, cluster *v2.Cluster) (Interface, error) {
	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			if hostIP == ip {
				if err := mergo.Merge(&host.SSH, &cluster.Spec.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&host.SSH, true), nil
			}
		}
	}
	return nil, fmt.Errorf("get host ssh client failed, host ip %s not in hosts ip list", hostIP)
}
