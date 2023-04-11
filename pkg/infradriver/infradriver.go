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
	"net"

	k8sv1 "k8s.io/api/core/v1"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

// InfraDriver treat the entire cluster as an operating system kernel,
// interface function here is the target system call.
type InfraDriver interface {
	// GetHostTaints GetHostTaint return host taint
	GetHostTaints(host net.IP) []k8sv1.Taint

	GetHostIPList() []net.IP

	GetHostIPListByRole(role string) []net.IP

	GetRoleListByHostIP(ip string) []string

	GetHostsPlatform(hosts []net.IP) (map[v1.Platform][]net.IP, error)

	//GetHostEnv return merged env with host env and cluster env.
	GetHostEnv(host net.IP) map[string]string

	//GetHostLabels return host labels.
	GetHostLabels(host net.IP) map[string]string

	//GetClusterEnv return cluster.spec.env as map[string]interface{}
	GetClusterEnv() map[string]string

	AddClusterEnv(envs []string)

	//GetClusterName ${clusterName}
	GetClusterName() string

	//GetClusterImageName ${cluster image Name}
	GetClusterImageName() string

	//GetClusterLaunchCmds ${user-defined launch command}
	GetClusterLaunchCmds() []string

	//GetClusterLaunchApps ${user-defined launch apps}
	GetClusterLaunchApps() []string

	//GetClusterRootfsPath /var/lib/sealer/data/${clusterName}/rootfs
	GetClusterRootfsPath() string

	// GetClusterBasePath /var/lib/sealer/data/${clusterName}
	GetClusterBasePath() string

	// Execute use eg.Go to execute shell cmd concurrently
	Execute(hosts []net.IP, f func(host net.IP) error) error

	// Copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host net.IP, localFilePath, remoteFilePath string) error
	// CopyR copy remote host files to localhost
	CopyR(host net.IP, remoteFilePath, localFilePath string) error
	// CmdAsync exec command on remote host, and asynchronous return logs
	CmdAsync(host net.IP, env map[string]string, cmd ...string) error
	// Cmd exec command on remote host, and return combined standard output and standard error
	Cmd(host net.IP, env map[string]string, cmd string) ([]byte, error)
	// CmdToString exec command on remote host, and return spilt standard output and standard error
	CmdToString(host net.IP, env map[string]string, cmd, spilt string) (string, error)

	// IsFileExist check remote file exist or not
	IsFileExist(host net.IP, remoteFilePath string) (bool, error)
	// IsDirExist Remote file existence returns true, nil
	IsDirExist(host net.IP, remoteDirPath string) (bool, error)

	// GetPlatform Get remote platform
	GetPlatform(host net.IP) (v1.Platform, error)

	GetHostName(host net.IP) (string, error)
	// Ping Ping remote host
	Ping(host net.IP) error
	// SetHostName add or update host name on host
	SetHostName(host net.IP, hostName string) error

	//SetClusterHostAliases set additional HostAliases
	SetClusterHostAliases(hosts []net.IP) error

	//DeleteClusterHostAliases delete additional HostAliases
	DeleteClusterHostAliases(hosts []net.IP) error

	GetClusterRegistry() v2.Registry
	// SetLvsRule add or update host name on host
	//SetLvsRule(host net.IP, hostName string) error
}
