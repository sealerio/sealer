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
	"net"
	"strings"

	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/ssh"
	"golang.org/x/sync/errgroup"
)

type SSHInfraDriver struct {
	sshConfigs        map[string]ssh.Interface
	hosts             []net.IP
	roleHostsMap      map[string][]net.IP
	hostEnvMap        map[string]map[string]interface{}
	clusterRenderData map[string]interface{}
	clusterName       string
	clusterImagerName string
}

func mergeList(hostEnv, globalEnv map[string]interface{}) map[string]interface{} {
	if len(hostEnv) == 0 {
		return globalEnv
	}
	for globalEnvKey, globalEnvValue := range globalEnv {
		if _, ok := hostEnv[globalEnvKey]; ok {
			continue
		}
		hostEnv[globalEnvKey] = globalEnvValue
	}
	return hostEnv
}

// ConvertEnv Convert []string to map[string]interface{}
func ConvertEnv(envList []string) (env map[string]interface{}) {
	temp := make(map[string][]string)
	env = make(map[string]interface{})

	for _, e := range envList {
		var kv []string
		if kv = strings.SplitN(e, "=", 2); len(kv) != 2 {
			continue
		}
		temp[kv[0]] = append(temp[kv[0]], strings.Split(kv[1], ";")...)
	}

	for k, v := range temp {
		if len(v) > 1 {
			env[k] = v
			continue
		}
		if len(v) == 1 {
			env[k] = v[0]
		}
	}

	return
}

func NewInfraDriver(cluster *v2.Cluster) (InfraDriver, error) {
	var err error
	ret := &SSHInfraDriver{
		clusterName:       cluster.Name,
		clusterImagerName: cluster.Spec.Image,
		sshConfigs:        map[string]ssh.Interface{},
		roleHostsMap:      map[string][]net.IP{},
		// todo need to separate env into app render data and sys render data
		hostEnvMap:        map[string]map[string]interface{}{},
		clusterRenderData: ConvertEnv(cluster.Spec.Env),
	}

	// initialize hosts field
	for _, host := range cluster.Spec.Hosts {
		ret.hosts = append(ret.hosts, host.IPS...)
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

	// initialize hostEnvMap field
	// merge the host ENV and global env, the host env will overwrite cluster.Spec.Env
	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			ret.hostEnvMap[ip.String()] = mergeList(ConvertEnv(host.Env), ret.clusterRenderData)
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

func (d *SSHInfraDriver) GetHostEnv(host net.IP) map[string]interface{} {
	return d.hostEnvMap[host.String()]
}

func (d *SSHInfraDriver) GetClusterRenderData() map[string]interface{} {
	return d.clusterRenderData
}

func (d *SSHInfraDriver) Copy(host net.IP, localFilePath, remoteFilePath string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.Copy(host, localFilePath, remoteFilePath)
}

func (d *SSHInfraDriver) CopyR(host net.IP, remoteFilePath, localFilePath string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	//client.CopyR take remoteFilePath as src file
	return client.CopyR(host, localFilePath, remoteFilePath)
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

func (d *SSHInfraDriver) GetClusterImageName() string {
	return d.clusterImagerName
}

func (d *SSHInfraDriver) GetHostName(hostIP net.IP) (string, error) {
	hostName, err := d.CmdToString(hostIP, "hostname", "")
	if err != nil {
		return "", err
	}
	if hostName == "" {
		return "", fmt.Errorf("faild to get remote hostname of host(%s)", hostIP.String())
	}

	return strings.ToLower(hostName), nil
}

func (d *SSHInfraDriver) GetHostsPlatform(hosts []net.IP) (map[v1.Platform][]net.IP, error) {
	hostsPlatformMap := make(map[v1.Platform][]net.IP)

	for _, ip := range hosts {
		plat, err := d.GetPlatform(ip)
		if err != nil {
			return nil, err
		}

		_, ok := hostsPlatformMap[plat]
		if !ok {
			hostsPlatformMap[plat] = []net.IP{ip}
		} else {
			hostsPlatformMap[plat] = append(hostsPlatformMap[plat], ip)
		}
	}

	return hostsPlatformMap, nil
}

func (d *SSHInfraDriver) GetClusterRootfs() string {
	return common.DefaultTheClusterRootfsDir(d.clusterName)
}

func (d *SSHInfraDriver) GetClusterBasePath() string {
	return common.DefaultClusterBaseDir(d.clusterName)
}

func (d *SSHInfraDriver) ConcurrencyExecute(hosts []net.IP, f func(host net.IP) error) error {
	eg, _ := errgroup.WithContext(context.Background())
	for _, ip := range hosts {
		host := ip
		eg.Go(func() error {
			err := f(host)
			if err != nil {
				return fmt.Errorf("on host [%s]: %v", host.String(), err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	return nil
}
