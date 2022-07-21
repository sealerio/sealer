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

package registry

import (
	"fmt"
	"net"
	"path/filepath"

	"github.com/sealerio/sealer/common"
	osi "github.com/sealerio/sealer/utils/os"
	"github.com/sealerio/sealer/utils/yaml"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	ConfigFile = "registry.yml"
	SeaHub     = "sea.hub"
)

type Config struct {
	IP       net.IP `yaml:"ip,omitempty"`
	Domain   string `yaml:"domain,omitempty"`
	Port     string `yaml:"port,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (c *Config) GenerateHtPasswd() (string, error) {
	if c.Username == "" || c.Password == "" {
		return "", fmt.Errorf("failed to generate htpasswd: registry username or passwodr is empty")
	}
	pwdHash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to generate registry password: %v", err)
	}
	return c.Username + ":" + string(pwdHash), nil
}

func (c *Config) Repo() string {
	return fmt.Sprintf("%s:%s", c.Domain, c.Port)
}

func GetConfig(rootfs string, defaultRegistryIP net.IP) *Config {
	var config Config
	var DefaultConfig = &Config{
		IP:     defaultRegistryIP,
		Domain: SeaHub,
		Port:   "5000",
	}
	registryConfigPath := filepath.Join(rootfs, common.EtcDir, ConfigFile)
	if !osi.IsFileExist(registryConfigPath) {
		logrus.Debug("use default registry config")
		return DefaultConfig
	}
	err := yaml.UnmarshalFile(registryConfigPath, &config)
	if err != nil {
		logrus.Errorf("failed to read registry config: %v", err)
		return DefaultConfig
	}
	if config.IP == nil {
		config.IP = DefaultConfig.IP
	}
	if config.Port == "" {
		config.Port = DefaultConfig.Port
	}
	if config.Domain == "" {
		config.Domain = DefaultConfig.Domain
	}
	logrus.Debugf("show registry info, IP: %s, Domain: %s", config.IP, config.Domain)
	return &config
}
