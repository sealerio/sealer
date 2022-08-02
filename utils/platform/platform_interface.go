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

package platform

import (
	"fmt"
	"net"
	"time"

	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	utilsnet "github.com/sealerio/sealer/utils/net"
	"github.com/sealerio/sealer/utils/os/fs"
	"github.com/sirupsen/logrus"
)

type Interface interface {
	// Platform Get remote platform
	Platform(host net.IP) (v1.Platform, error)
}

func GetClusterPlatform(cluster *v2.Cluster) (map[string]v1.Platform, error) {
	clusterStatus := make(map[string]v1.Platform)
	for _, ip := range cluster.GetAllIPList() {
		IP := ip
		ssh, err := GetHostSSHClient(IP, cluster)
		//ssh, err := platform.GetHostSSHClient(IP, cluster)
		if err != nil {
			return nil, err
		}
		clusterStatus[IP.String()], err = ssh.Platform(IP)
		if err != nil {
			return nil, err
		}
	}
	return clusterStatus, nil
}

func NewSSHClient(ssh *v1.SSH, isStdout bool) Interface {
	if ssh.User == "" {
		ssh.User = common.ROOT
	}
	address, err := utilsnet.GetLocalHostAddresses()
	if err != nil {
		logrus.Warnf("failed to get local address: %v", err)
	}

	return &RemotePlatform{
		sshRemote: struct {
			IsStdout     bool
			Encrypted    bool
			User         string
			Password     string
			Port         string
			PkFile       string
			PkPassword   string
			Timeout      *time.Duration
			LocalAddress []net.Addr
			Fs           fs.Interface
		}{
			IsStdout:     isStdout,
			Encrypted:    ssh.Encrypted,
			User:         ssh.User,
			Password:     ssh.Passwd,
			Port:         ssh.Port,
			PkFile:       ssh.Pk,
			PkPassword:   ssh.PkPasswd,
			LocalAddress: address,
			Fs:           fs.NewFilesystem(),
		},
	}
}

func GetHostSSHClient(hostIP net.IP, cluster *v2.Cluster) (Interface, error) {
	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			if hostIP.Equal(ip) {
				if err := mergo.Merge(&host.SSH, &cluster.Spec.SSH); err != nil {
					return nil, err
				}
				return NewSSHClient(&host.SSH, false), nil
			}
		}
	}
	return nil, fmt.Errorf("failed to get host ssh client: host ip %s not in hosts ip list", hostIP)
}
