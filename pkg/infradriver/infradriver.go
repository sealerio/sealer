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

package infradriver

import (
	v1 "github.com/sealerio/sealer/types/api/v1"
	"net"
)

// InfraDriver treat the entire cluster as an operating system kernel,
// interface function here is the target system call.
type InfraDriver interface {
	GetHostIPList() []net.IP

	GetHostIPListByRole(role string) []net.IP

	//GetClusterName ${clusterName}
	GetClusterName() string

	//GetClusterRootfs /var/lib/sealer/data/${clusterName}/rootfs
	GetClusterRootfs() string

	// GetImageMountDir /var/lib/sealer/data/${clusterName}/mount
	GetImageMountDir() string

	// GetClusterBasePath /var/lib/sealer/data/${clusterName}
	GetClusterBasePath() string

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
