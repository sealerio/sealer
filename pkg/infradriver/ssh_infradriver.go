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
	"path/filepath"
	"strings"

	"github.com/containers/buildah/util"
	"github.com/imdario/mergo"
	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	mapUtils "github.com/sealerio/sealer/utils/maps"
	"github.com/sealerio/sealer/utils/shellcommand"
	"github.com/sealerio/sealer/utils/ssh"
	strUtil "github.com/sealerio/sealer/utils/strings"
	"golang.org/x/sync/errgroup"
	k8sv1 "k8s.io/api/core/v1"
	k8snet "k8s.io/utils/net"
)

type SSHInfraDriver struct {
	sshConfigs   map[string]ssh.Interface
	hosts        []net.IP
	hostTaint    map[string][]k8sv1.Taint
	hostRolesMap map[string][]string
	roleHostsMap map[string][]net.IP
	hostLabels   map[string]map[string]string
	hostEnvMap   map[string]map[string]string
	clusterEnv   map[string]string
	cluster      v2.Cluster
}

func convertTaints(taints []string) ([]k8sv1.Taint, error) {
	var k8staints []k8sv1.Taint
	for _, taint := range taints {
		data, err := formatData(taint)
		if err != nil {
			return nil, err
		}
		k8staints = append(k8staints, data)
	}
	return k8staints, nil
}

// NewInfraDriver will create a new Infra driver, and if extraEnv specified, it will set env not exist in Cluster
func NewInfraDriver(cluster *v2.Cluster) (InfraDriver, error) {
	var err error
	ret := &SSHInfraDriver{
		cluster:      *cluster,
		sshConfigs:   map[string]ssh.Interface{},
		roleHostsMap: map[string][]net.IP{},
		hostRolesMap: map[string][]string{},
		// todo need to separate env into app render data and sys render data
		hostEnvMap: map[string]map[string]string{},
		hostLabels: map[string]map[string]string{},
		hostTaint:  map[string][]k8sv1.Taint{},
	}

	// initialize hosts field
	for _, host := range cluster.Spec.Hosts {
		ret.hosts = append(ret.hosts, host.IPS...)
	}

	if len(ret.hosts) == 0 {
		return nil, fmt.Errorf("no hosts specified")
	}

	if err = checkAllHostsSameFamily(ret.hosts); err != nil {
		return nil, err
	}

	if k8snet.IsIPv6String(ret.hosts[0].String()) {
		hostIPFamilyEnv := fmt.Sprintf("%s=%s", common.EnvHostIPFamily, k8snet.IPv6)
		if !util.StringInSlice(hostIPFamilyEnv, cluster.Spec.Env) {
			cluster.Spec.Env = append(cluster.Spec.Env, hostIPFamilyEnv)
		}
	}

	// initialize sshConfigs field
	for i := range cluster.Spec.Hosts {
		if err = mergo.Merge(&cluster.Spec.Hosts[i].SSH, &cluster.Spec.SSH); err != nil {
			return nil, err
		}
		for _, ip := range cluster.Spec.Hosts[i].IPS {
			ret.sshConfigs[ip.String()] = ssh.NewSSHClient(&cluster.Spec.Hosts[i].SSH, true)
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
		for _, ip := range host.IPS {
			ret.hostRolesMap[ip.String()] = host.Roles
		}
	}

	ret.clusterEnv = strUtil.ConvertStringSliceToMap(cluster.Spec.Env)

	// initialize hostEnvMap and host labels field
	// merge the host ENV and global env, the host env will overwrite cluster.Spec.Env
	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			ret.hostEnvMap[ip.String()] = mapUtils.Merge(strUtil.ConvertStringSliceToMap(host.Env), ret.clusterEnv)
			ret.hostLabels[ip.String()] = host.Labels
		}
	}

	for _, host := range cluster.Spec.Hosts {
		for _, ip := range host.IPS {
			ret.hostTaint[ip.String()], err = convertTaints(host.Taints)
			if err != nil {
				return nil, err
			}
		}
	}

	return ret, err
}

func (d *SSHInfraDriver) GetHostTaints(host net.IP) []k8sv1.Taint {
	return d.hostTaint[host.String()]
}

func (d *SSHInfraDriver) GetHostIPList() []net.IP {
	return d.hosts
}

func (d *SSHInfraDriver) GetHostIPListByRole(role string) []net.IP {
	return d.roleHostsMap[role]
}

func (d *SSHInfraDriver) GetRoleListByHostIP(ip string) []string {
	return d.hostRolesMap[ip]
}

func (d *SSHInfraDriver) GetHostEnv(host net.IP) map[string]string {
	// Set env for each host
	hostEnv := d.hostEnvMap[host.String()]
	if _, ok := hostEnv[common.EnvHostIP]; !ok {
		hostEnv[common.EnvHostIP] = host.String()
	}
	return hostEnv
}

func (d *SSHInfraDriver) GetHostLabels(host net.IP) map[string]string {
	return d.hostLabels[host.String()]
}

func (d *SSHInfraDriver) GetClusterEnv() map[string]string {
	return d.clusterEnv
}

func (d *SSHInfraDriver) AddClusterEnv(envs []string) {
	if d.clusterEnv == nil || envs == nil {
		return
	}
	newEnv := strUtil.ConvertStringSliceToMap(envs)
	for k, v := range newEnv {
		d.clusterEnv[k] = v
	}
}

func (d *SSHInfraDriver) GetClusterRegistry() v2.Registry {
	return d.cluster.Spec.Registry
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

func (d *SSHInfraDriver) CmdAsync(host net.IP, env map[string]string, cmd ...string) error {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.CmdAsync(host, env, cmd...)
}

func (d *SSHInfraDriver) Cmd(host net.IP, env map[string]string, cmd string) ([]byte, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return nil, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.Cmd(host, env, cmd)
}

func (d *SSHInfraDriver) CmdToString(host net.IP, env map[string]string, cmd, spilt string) (string, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return "", fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.CmdToString(host, env, cmd, spilt)
}

func (d *SSHInfraDriver) IsFileExist(host net.IP, remoteFilePath string) (bool, error) {
	client := d.sshConfigs[host.String()]
	if client == nil {
		return false, fmt.Errorf("ip(%s) is not in cluster", host.String())
	}
	return client.IsFileExist(host, remoteFilePath)
}

func (d *SSHInfraDriver) IsDirExist(host net.IP, remoteDirPath string) (bool, error) {
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
	return d.CmdAsync(host, nil, setHostNameCmd)
}

func (d *SSHInfraDriver) SetClusterHostAliases(hosts []net.IP) error {
	for _, host := range hosts {
		for _, hostAliases := range d.cluster.Spec.HostAliases {
			hostname := strings.Join(hostAliases.Hostnames, " ")
			err := d.CmdAsync(host, nil, shellcommand.CommandSetHostAlias(hostname, hostAliases.IP))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (d *SSHInfraDriver) DeleteClusterHostAliases(hosts []net.IP) error {
	for _, host := range hosts {
		err := d.CmdAsync(host, nil, shellcommand.CommandUnSetHostAlias())
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *SSHInfraDriver) GetClusterName() string {
	return d.cluster.Name
}

func (d *SSHInfraDriver) GetClusterImageName() string {
	return d.cluster.Spec.Image
}

func (d *SSHInfraDriver) GetClusterLaunchCmds() []string {
	return d.cluster.Spec.CMD
}

func (d *SSHInfraDriver) GetClusterLaunchApps() []string {
	return d.cluster.Spec.APPNames
}

func (d *SSHInfraDriver) GetHostName(hostIP net.IP) (string, error) {
	hostName, err := d.CmdToString(hostIP, nil, "uname -n", "")
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

func (d *SSHInfraDriver) GetClusterRootfsPath() string {
	dataRoot := d.cluster.Spec.DataRoot
	if dataRoot == "" {
		dataRoot = common.DefaultSealerDataDir
	}

	return filepath.Join(dataRoot, d.cluster.Name, "rootfs")
}

func (d *SSHInfraDriver) GetClusterBasePath() string {
	dataRoot := d.cluster.Spec.DataRoot
	if dataRoot == "" {
		dataRoot = common.DefaultSealerDataDir
	}

	return filepath.Join(dataRoot, d.cluster.Name)
}

func (d *SSHInfraDriver) Execute(hosts []net.IP, f func(host net.IP) error) error {
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

func checkAllHostsSameFamily(nodeList []net.IP) error {
	var netFamily bool
	for i, ip := range nodeList {
		if i == 0 {
			netFamily = k8snet.IsIPv4(ip)
		}

		if netFamily != k8snet.IsIPv4(ip) {
			return fmt.Errorf("all hosts must be in same ip family, but the node list given are mixed with ipv4 and ipv6: %v", nodeList)
		}
	}
	return nil
}
