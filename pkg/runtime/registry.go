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

package runtime

import (
	"fmt"
	"path/filepath"

	"github.com/alibaba/sealer/common"

	"github.com/alibaba/sealer/logger"
	"github.com/alibaba/sealer/utils"
	"golang.org/x/crypto/bcrypt"
)

const (
	RegistryName                = "sealer-registry"
	RegistryBindDest            = "/var/lib/registry"
	RegistryBindConfig          = "registry_config.yml"
	RegistryCustomConfig        = "registry.yml"
	SeaHub                      = "sea.hub"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DockerLoginCommand          = "docker login %s -u %s -p %s && " + KubeletAuthCommand
	KubeletAuthCommand          = "cp /root/.docker/config.json /var/lib/kubelet && systemctl restart kubelet"
)

type RegistryConfig struct {
	IP       string `yaml:"ip,omitempty"`
	Domain   string `yaml:"domain,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (k *KubeadmRuntime) getRegistryHost() (host string) {
	ip, _ := utils.GetSSHHostIPAndPort(k.RegConfig.IP)
	return fmt.Sprintf("%s %s", ip, k.RegConfig.Domain)
}

// ApplyRegistry Only use this for join and init, due to the initiation operations.
func (k *KubeadmRuntime) ApplyRegistry() error {
	ssh, err := k.getHostSSHClient(k.RegConfig.IP)
	if err != nil {
		return fmt.Errorf("failed to get registry ssh client: %v", err)
	}

	if k.RegConfig.Username != "" && k.RegConfig.Password != "" {
		htpasswd, err := k.RegConfig.GenerateHtPasswd()
		if err != nil {
			return err
		}
		err = ssh.CmdAsync(k.RegConfig.IP, fmt.Sprintf("echo '%s' > %s", htpasswd, filepath.Join(k.getRootfs(), "etc", DefaultRegistryHtPasswdFile)))
		if err != nil {
			return err
		}
	}
	initRegistry := fmt.Sprintf("cd %s/scripts && sh init-registry.sh %s %s %s", k.getRootfs(), k.RegConfig.Port, fmt.Sprintf("%s/registry", k.getRootfs()), k.RegConfig.Domain)
	registryHost := k.getRegistryHost()
	addRegistryHosts := fmt.Sprintf(RemoteAddEtcHosts, registryHost, registryHost)
	if k.RegConfig.Domain != SeaHub {
		addSeaHubHosts := fmt.Sprintf(RemoteAddEtcHosts, k.RegConfig.IP+" "+SeaHub, k.RegConfig.IP+" "+SeaHub)
		addRegistryHosts = fmt.Sprintf("%s && %s", addRegistryHosts, addSeaHubHosts)
	}
	if err = ssh.CmdAsync(k.RegConfig.IP, initRegistry); err != nil {
		return err
	}
	if err = ssh.CmdAsync(k.GetMaster0IP(), addRegistryHosts); err != nil {
		return err
	}
	if k.RegConfig.Username == "" || k.RegConfig.Password == "" {
		return nil
	}
	return ssh.CmdAsync(k.GetMaster0IP(), fmt.Sprintf(DockerLoginCommand, k.RegConfig.Domain+":"+k.RegConfig.Port, k.RegConfig.Username, k.RegConfig.Password))
}

func (r *RegistryConfig) GenerateHtPasswd() (string, error) {
	if r.Username == "" || r.Password == "" {
		return "", fmt.Errorf("generate htpasswd failed: registry username or passwodr is empty")
	}
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(r.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate registry password: %v", err)
	}
	return r.Username + ":" + string(pwdHash), nil
}

func (r *RegistryConfig) Repo() string {
	return fmt.Sprintf("%s:%s", r.Domain, r.Port)
}

func GetRegistryConfig(rootfs, defaultRegistry string) *RegistryConfig {
	var config RegistryConfig
	var DefaultConfig = &RegistryConfig{
		IP:     defaultRegistry,
		Domain: SeaHub,
		Port:   "5000",
	}
	registryConfigPath := filepath.Join(rootfs, common.EtcDir, RegistryCustomConfig)
	if !utils.IsFileExist(registryConfigPath) {
		logger.Debug("use default registry config")
		return DefaultConfig
	}
	err := utils.UnmarshalYamlFile(registryConfigPath, &config)
	if err != nil {
		logger.Error("Failed to read registry config! ")
		return DefaultConfig
	}
	if config.IP == "" {
		config.IP = DefaultConfig.IP
	}
	if config.Port == "" {
		config.Port = DefaultConfig.Port
	}
	if config.Domain == "" {
		config.Domain = DefaultConfig.Domain
	}
	logger.Debug("show registry info, IP: %s, Domain: %s", config.IP, config.Domain)
	return &config
}

func (k *KubeadmRuntime) DeleteRegistry() error {
	ssh, err := k.getHostSSHClient(k.RegConfig.IP)
	if err != nil {
		return fmt.Errorf("failed to delete registry: %v", err)
	}

	cmd := fmt.Sprintf("if docker inspect %s;then docker rm -f %s;fi", RegistryName, RegistryName)
	return ssh.CmdAsync(k.RegConfig.IP, cmd)
}
