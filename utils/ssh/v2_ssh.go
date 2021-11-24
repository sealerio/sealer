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
	"sync"
	"time"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v2 "github.com/alibaba/sealer/types/api/v2"
	"github.com/alibaba/sealer/utils"
	"github.com/imdario/mergo"
)

type V2Interface interface {
	// copy local files to remote host
	// scp -r /tmp root@192.168.0.2:/root/tmp => Copy("192.168.0.2","tmp","/root/tmp")
	// need check md5sum
	Copy(host, srcFilePath, dstFilePath string) error
	// copy remote host files to localhost
	Fetch(host, srcFilePath, dstFilePath string) error
	// exec command on remote host, and asynchronous return logs
	CmdAsync(host string, cmd ...string) error
	// exec command on remote host, and return combined standard output and standard error
	Cmd(host, cmd string) ([]byte, error)
	// check remote file exist or not
	IsFileExist(host, remoteFilePath string) bool
	//Remote file existence returns true, nil
	RemoteDirExist(host, remoteDirpath string) (bool, error)
	// exec command on remote host, and return spilt standard output and standard error
	CmdToString(host, cmd, spilt string) (string, error)
	Ping(host string) error
}

type V2SSH struct {
	hosts        map[string]Interface
	LocalAddress *[]net.Addr
}

func NewV2SSHByCluster(cluster *v2.Cluster) V2Interface {
	if cluster.Spec.SSH.User == "" {
		cluster.Spec.SSH.User = common.ROOT
	}
	address, err := utils.IsLocalHostAddrs()
	if err != nil {
		logger.Warn("failed to get local address, %v", err)
	}
	defaultSSH := cluster.Spec.SSH
	hostsMap := make(map[string]Interface)
	for i := range cluster.Spec.Hosts {
		host := cluster.Spec.Hosts[i]
		//use host ssh override the default ssh
		err := mergo.Merge(&host.SSH, defaultSSH, mergo.WithOverride)
		if err != nil {
			logger.Error("failed to merge default ssh, err: %v", err)
		}
		sshClient := NewSSHClient(host.SSH)
		for _, ip := range host.IPS {
			hostsMap[ip] = sshClient
		}
	}
	return &V2SSH{
		hosts:        hostsMap,
		LocalAddress: address,
	}
}

type V2Client struct {
	SSH  V2Interface
	Host string
}

func WaitV2SSHReady(ssh V2Interface, tryTimes int, hosts ...string) error {
	var err error
	var wg sync.WaitGroup
	for _, ip := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()
			for i := 0; i < tryTimes; i++ {
				err = ssh.Ping(host)
				if err == nil {
					return
				}
				time.Sleep(time.Duration(i) * time.Second)
			}
			err = fmt.Errorf("wait for [%s] ssh ready timeout:  %v, ensure that the IP address or password is correct", host, err)
		}(ip)
	}
	wg.Wait()
	return err
}

func (v V2SSH) Copy(host, srcFilePath, dstFilePath string) error {
	return v.hosts[host].Copy(host, srcFilePath, dstFilePath)
}

func (v V2SSH) Fetch(host, srcFilePath, dstFilePath string) error {
	return v.hosts[host].Fetch(host, srcFilePath, dstFilePath)
}

func (v V2SSH) CmdAsync(host string, cmd ...string) error {
	return v.hosts[host].CmdAsync(host, cmd...)
}

func (v V2SSH) Cmd(host, cmd string) ([]byte, error) {
	return v.hosts[host].Cmd(host, cmd)
}

func (v V2SSH) IsFileExist(host, remoteFilePath string) bool {
	return v.hosts[host].IsFileExist(host, remoteFilePath)
}

func (v V2SSH) RemoteDirExist(host, remoteDirpath string) (bool, error) {
	return v.hosts[host].RemoteDirExist(host, remoteDirpath)
}

func (v V2SSH) CmdToString(host, cmd, spilt string) (string, error) {
	return v.hosts[host].CmdToString(host, cmd, spilt)
}

func (v V2SSH) Ping(host string) error {
	return v.hosts[host].Ping(host)
}
