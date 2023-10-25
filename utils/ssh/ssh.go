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
	"github.com/sirupsen/logrus"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	netUtils "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/os/fs"
)

type Interface interface {
	// Copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host net.IP, srcFilePath, dstFilePath string) error
	// CopyR copy remote host files to localhost
	CopyR(host net.IP, srcFilePath, dstFilePath string) error
	// CmdAsync exec command on remote host, and asynchronous return logs
	CmdAsync(host net.IP, env map[string]string, cmd ...string) error
	// Cmd exec command on remote host, and return combined standard output and standard error
	Cmd(host net.IP, env map[string]string, cmd string) ([]byte, error)
	// CmdToString exec command on remote host, and return spilt standard output and standard error
	CmdToString(host net.IP, env map[string]string, cmd, spilt string) (string, error)
	// IsFileExist check remote file exist or not
	IsFileExist(host net.IP, remoteFilePath string) (bool, error)
	// RemoteDirExist Remote file existence returns true, nil
	RemoteDirExist(host net.IP, remoteDirpath string) (bool, error)
	// GetPlatform Get remote platform
	GetPlatform(host net.IP) (v1.Platform, error)
	// Ping Ping remote host
	Ping(host net.IP) error
}

type SSH struct {
	AlsoToStdout bool
	Encrypted    bool
	User         string
	Password     string
	Port         string
	PkFile       string
	PkPassword   string
	Timeout      *time.Duration
	LocalAddress []net.Addr
	Fs           fs.Interface
}

func NewSSHClient(ssh *v1.SSH, alsoToStdout bool) Interface {
	if ssh.User == "" {
		ssh.User = common.ROOT
	}
	address, err := netUtils.GetLocalHostAddresses()
	if err != nil {
		logrus.Warnf("failed to get local address: %v", err)
	}
	return &SSH{
		AlsoToStdout: alsoToStdout,
		Encrypted:    ssh.Encrypted,
		User:         ssh.User,
		Password:     ssh.Passwd,
		Port:         ssh.Port,
		PkFile:       ssh.Pk,
		PkPassword:   ssh.PkPasswd,
		LocalAddress: address,
		Fs:           fs.NewFilesystem(),
	}
}

// GetHostSSHClient is used to executed bash command and no std out to be printed.
func GetHostSSHClient(hostIP net.IP, cluster *v2.Cluster) (Interface, error) {
	for i := range cluster.Spec.Hosts {
		for _, ip := range cluster.Spec.Hosts[i].IPS {
			if hostIP.Equal(ip) {
				if err := mergo.Merge(&cluster.Spec.Hosts[i].SSH, &cluster.Spec.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&cluster.Spec.Hosts[i].SSH, false), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to get host ssh client: host ip %s not in hosts ip list", hostIP)
}

// NewStdoutSSHClient is used to show std out when execute bash command.
func NewStdoutSSHClient(hostIP net.IP, cluster *v2.Cluster) (Interface, error) {
	for i := range cluster.Spec.Hosts {
		for _, ip := range cluster.Spec.Hosts[i].IPS {
			if hostIP.Equal(ip) {
				if err := mergo.Merge(&cluster.Spec.Hosts[i].SSH, &cluster.Spec.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&cluster.Spec.Hosts[i].SSH, true), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to get host ssh client: host ip %s not in hosts ip list", hostIP)
}
