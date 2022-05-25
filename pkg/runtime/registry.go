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

	"github.com/sealerio/sealer/utils/yaml"

	osi "github.com/sealerio/sealer/utils/os"

	"github.com/sealerio/sealer/utils/net"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	"github.com/sealerio/sealer/pkg/cert"
	"golang.org/x/crypto/bcrypt"
)

const (
	RegistryName                = "sealer-registry"
	RegistryBindDest            = "/var/lib/registry"
	RegistryBindConfig          = "registry_config.yml"
	RegistryCustomConfig        = "registry.yml"
	SeaHub                      = "sea.hub"
	DefaultRegistryHtPasswdFile = "registry_htpasswd"
	DockerLoginCommand          = "nerdctl login -u %s -p %s %s && " + KubeletAuthCommand
	KubeletAuthCommand          = "cp /root/.docker/config.json /var/lib/kubelet"
)

type RegistryConfig struct {
	IP       string `yaml:"ip,omitempty"`
	Domain   string `yaml:"domain,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (k *KubeadmRuntime) getRegistryHost() (host string) {
	ip, _ := net.GetSSHHostIPAndPort(k.RegConfig.IP)
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
	return ssh.CmdAsync(k.GetMaster0IP(), k.GerLoginCommand())
}

func (k *KubeadmRuntime) GerLoginCommand() string {
	return fmt.Sprintf("%s && %s",
		fmt.Sprintf(DockerLoginCommand, k.RegConfig.Username, k.RegConfig.Password, k.RegConfig.Domain+":"+k.RegConfig.Port),
		fmt.Sprintf(DockerLoginCommand, k.RegConfig.Username, k.RegConfig.Password, SeaHub+":"+k.RegConfig.Port))
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
	if !osi.IsFileExist(registryConfigPath) {
		logger.Debug("use default registry config")
		return DefaultConfig
	}
	err := yaml.UnmarshalFile(registryConfigPath, &config)
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

	cmd := fmt.Sprintf("if docker inspect %s;then docker rm -f %[1]s;fi && ((! nerdctl ps -a |grep %[1]s) || (nerdctl stop %[1]s && nerdctl rmi -f %[1]s))", RegistryName)
	return ssh.CmdAsync(k.RegConfig.IP, cmd)
}

func GenerateRegistryCert(registryCertPath string, BaseName string) error {
	regCertConfig := cert.Config{
		Path:         registryCertPath,
		BaseName:     BaseName,
		CommonName:   BaseName,
		DNSNames:     []string{BaseName},
		Organization: []string{common.ExecBinaryFileName},
		Year:         100,
	}
	if BaseName != SeaHub {
		regCertConfig.DNSNames = append(regCertConfig.DNSNames, SeaHub)
	}
	crt, key, err := cert.NewCaCertAndKey(regCertConfig)
	if err != nil {
		return err
	}
	return cert.WriteCertAndKey(regCertConfig.Path, regCertConfig.BaseName, crt, key)
}
