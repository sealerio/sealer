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

package kubernetes

import (
	"fmt"
	"path/filepath"
)

const (
	RegistryName                = "sealer-registry"
	SeaHub                      = "sea.hub"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DockerLoginCommand          = "nerdctl login -u %s -p %s %s && " + KubeletAuthCommand
	KubeletAuthCommand          = "mkdir -p /var/lib/kubelet && cp /root/.docker/config.json /var/lib/kubelet"
	DeleteRegistryCommand       = "if docker inspect %s 2>/dev/null;then docker rm -f %[1]s;fi && ((! nerdctl ps -a 2>/dev/null |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))"
)

func (k *Runtime) addRegistryDomainToHosts() (host string) {
	content := fmt.Sprintf("%s %s", k.RegConfig.IP.String(), k.RegConfig.Domain)
	return fmt.Sprintf(RemoteAddEtcHosts, content, content)
}

// ApplyRegistry Only use this for join and init, due to the initiation operations.
func (k *Runtime) ApplyRegistry() error {
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
		fmt.Sprintf(DockerLoginCommand, k.RegConfig.Username, k.RegConfig.Password, k.RegConfig.Domain+":"+k.RegConfig.Port),
		fmt.Sprintf(DockerLoginCommand, k.RegConfig.Username, k.RegConfig.Password, SeaHub+":"+k.RegConfig.Port))
}

func (k *Runtime) DeleteRegistry() error {
	ssh, err := k.getHostSSHClient(k.RegConfig.IP)
	if err != nil {
		return fmt.Errorf("failed to delete registry: %v", err)
	}

	return ssh.CmdAsync(k.RegConfig.IP, fmt.Sprintf(DeleteRegistryCommand, RegistryName))
}
