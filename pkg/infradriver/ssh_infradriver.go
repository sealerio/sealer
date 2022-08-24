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
	"context"
	"fmt"
	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"
	"golang.org/x/sync/errgroup"
	"net"
	"path/filepath"
)

type SSHInfraDriver struct {
	sshConfigs   map[string]ssh.Interface
	hosts        []net.IP
	roleHostsMap map[string][]net.IP
	clusterName  string
}

func NewInfraDriver(cluster *v2.Cluster) (InfraDriver, error) {
	var err error
	ret := &SSHInfraDriver{
		clusterName:  cluster.Name,
		sshConfigs:   map[string]ssh.Interface{},
		hosts:        cluster.GetAllIPList(),
		roleHostsMap: map[string][]net.IP{},
	}

	// initialize sshConfigs field
	for _, host := range cluster.Spec.Hosts {
		if err = mergo.Merge(&host.SSH, &cluster.Spec.SSH); err != nil {
			return nil, err
		}
		for _, ip := range host.IPS {
			ret.sshConfigs[ip.String()] = ssh.NewSSHClient(&host.SSH, true)
		}
	}

	// initialize roleHostsMap field
	for _, host := range cluster.Spec.Hosts {
		for _, role := range host.Roles {
			ips, ok := ret.roleHostsMap[role]
			if !ok {
				ret.roleHostsMap[role] = host.IPS
			} else {
				ret.roleHostsMap[role] = append(ips, host.IPS...)
			}
		}
	}

	return ret, err
}

func (d *SSHInfraDriver) GetHostIPList() []net.IP {
	return d.hosts
}

func (d *SSHInfraDriver) GetHostIPListByRole(role string) []net.IP {
	return d.roleHostsMap[role]
}

func (d *SSHInfraDriver) Copy(host net.IP, srcFilePath, dstFilePath string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.Copy(host, srcFilePath, dstFilePath)
}

func (d *SSHInfraDriver) CopyR(host net.IP, srcFilePath, dstFilePath string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.CopyR(host, srcFilePath, dstFilePath)
}

func (d *SSHInfraDriver) CmdAsync(host net.IP, cmd ...string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.CmdAsync(host, cmd...)
}

func (d *SSHInfraDriver) Cmd(host net.IP, cmd string) ([]byte, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return nil, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.Cmd(host, cmd)
}

func (d *SSHInfraDriver) CmdToString(host net.IP, cmd, spilt string) (string, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return "", fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.CmdToString(host, cmd, spilt)
}

func (d *SSHInfraDriver) IsFileExist(host net.IP, remoteFilePath string) (bool, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return false, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.IsFileExist(host, remoteFilePath)
}

func (d *SSHInfraDriver) RemoteDirExist(host net.IP, remoteDirPath string) (bool, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return false, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.RemoteDirExist(host, remoteDirPath)
}

func (d *SSHInfraDriver) GetPlatform(host net.IP) (v1.Platform, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return v1.Platform{}, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.GetPlatform(host)
}

func (d *SSHInfraDriver) Ping(host net.IP) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.Ping(host)
}

func (d *SSHInfraDriver) SetHostName(host net.IP, hostName string) error {
	setHostNameCmd := fmt.Sprintf("hostnamectl set-hostname %s", hostName)
	return d.CmdAsync(host, setHostNameCmd)
}

func (d *SSHInfraDriver) GetClusterName() string {
	return d.clusterName
}

func (d *SSHInfraDriver) GetImageMountDir(platform v1.Platform) string {
	platPath := fmt.Sprintf("%s_%s_%s", platform.OS, platform.Architecture, platform.Variant)
	return filepath.Join(d.GetClusterRootfs(), "mount", platPath)
}

func (d *SSHInfraDriver) GetClusterRootfs() string {
	return common.DefaultTheClusterRootfsDir(d.clusterName)
}

func (d *SSHInfraDriver) GetClusterBasePath() string {
	return common.DefaultClusterBaseDir(d.clusterName)
}

func (d *SSHInfraDriver) ConcurrencyExecute(f func(host net.IP) error) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range d.hosts {
		host := ip
		eg.Go(func() error {
			err := f(host)
			if err != nil {
				return fmt.Errorf("on host [%s]: %v", ip.String(), err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
