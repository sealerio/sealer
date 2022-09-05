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

package k0s

import (
	"fmt"
	"net"
	"path/filepath"
)

const (
	SeaHub                      = "sea.hub"
	RemoteAddEtcHosts           = "cat /etc/hosts |grep '%s' || echo '%s' >> /etc/hosts"
	ContainerdLoginCommand      = "nerdctl login -u %s -p %s %s"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DockerCertDir               = "/etc/docker/certs.d"
	DeleteRegistryCommand       = "((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
	RegistryName                = "sealer-registry"
)

// sendRegistryCertAndKey send registry cert to Master0 host. path like: /var/lib/sealer/data/my-k0s-cluster/certs
func (k *Runtime) sendRegistryCertAndKey() error {
	return k.sendFileToHosts(k.cluster.GetMasterIPList()[:1], k.getCertsDir(), filepath.Join(k.getRootfs(), "certs"))
}

// sendRegistryCert send registry cert to host.
func (k *Runtime) sendRegistryCert(host []net.IP) error {
	cf := k.RegConfig
	err := k.sendFileToHosts(host, fmt.Sprintf("%s/%s.crt", k.getCertsDir(), cf.Domain), fmt.Sprintf("%s/%s:%s/%s.crt", DockerCertDir, cf.Domain, cf.Port, cf.Domain))
	if err != nil {
		return err
	}
	return k.sendFileToHosts(host, fmt.Sprintf("%s/%s.crt", k.getCertsDir(), cf.Domain), fmt.Sprintf("%s/%s:%s/%s.crt", DockerCertDir, SeaHub, cf.Port, cf.Domain))
}

func (k *Runtime) addRegistryDomainToHosts() (host string) {
	content := fmt.Sprintf("%s %s", k.RegConfig.IP.String(), k.RegConfig.Domain)
	return fmt.Sprintf(RemoteAddEtcHosts, content, content)
}

// ApplyRegistryOnMaster0 Only use this for init, due to the initiation operations.
func (k *Runtime) ApplyRegistryOnMaster0() error {
	ssh, err := k.getHostSSHClient(k.RegConfig.IP)
	if err != nil {
		return fmt.Errorf("failed to get registry ssh client: %v", err)
	}

	if k.RegConfig.Username != "" && k.RegConfig.Password != "" {
		htpasswd, err := k.RegConfig.GenerateHTTPBasicAuth()
		if err != nil {
			return err
		}
		err = ssh.CmdAsync(k.RegConfig.IP, fmt.Sprintf("echo '%s' > %s", htpasswd, filepath.Join(k.getRootfs(), "etc", DefaultRegistryHtPasswdFile)))
		if err != nil {
			return err
		}
	}
	initRegistry := fmt.Sprintf("cd %s/scripts && ./init-registry.sh %s %s %s", k.getRootfs(), k.RegConfig.Port, fmt.Sprintf("%s/registry", k.getRootfs()), k.RegConfig.Domain)
	addRegistryHosts := k.addRegistryDomainToHosts()
	if k.RegConfig.Domain != SeaHub {
		addSeaHubHosts := fmt.Sprintf(RemoteAddEtcHosts, k.RegConfig.IP.String()+" "+SeaHub, k.RegConfig.IP.String()+" "+SeaHub)
		addRegistryHosts = fmt.Sprintf("%s && %s", addRegistryHosts, addSeaHubHosts)
	}
	if err = ssh.CmdAsync(k.RegConfig.IP, initRegistry); err != nil {
		return err
	}
	if err = ssh.CmdAsync(k.cluster.GetMaster0IP(), addRegistryHosts); err != nil {
		return err
	}
	if k.RegConfig.Username == "" || k.RegConfig.Password == "" {
		return nil
	}
	return ssh.CmdAsync(k.cluster.GetMaster0IP(), k.GenLoginCommand())
}

func (k *Runtime) GenLoginCommand() string {
	return fmt.Sprintf("%s && %s",
		fmt.Sprintf(ContainerdLoginCommand, k.RegConfig.Username, k.RegConfig.Password, k.RegConfig.Domain+":"+k.RegConfig.Port),
		fmt.Sprintf(ContainerdLoginCommand, k.RegConfig.Username, k.RegConfig.Password, SeaHub+":"+k.RegConfig.Port))
}

func (k *Runtime) GenerateRegistryCert() error {
	return GenerateRegistryCert(k.getCertsDir(), k.RegConfig.Domain)
}

func (k *Runtime) SendRegistryCert(host []net.IP) error {
	err := k.sendRegistryCertAndKey()
	if err != nil {
		return err
	}
	return k.sendRegistryCert(host)
}

func (k *Runtime) DeleteRegistry() error {
	ssh, err := k.getHostSSHClient(k.RegConfig.IP)
	if err != nil {
		return fmt.Errorf("failed to delete registry: %v", err)
	}

	return ssh.CmdAsync(k.RegConfig.IP, fmt.Sprintf(DeleteRegistryCommand, RegistryName))
}
