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

func (c *Config) GenerateHTTPBasicAuth() (string, error) {
	if c.Username == "" || c.Password == "" {
		return "", fmt.Errorf("failed to generate HTTP basic authentication: registry username or password is empty")
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

func GetConfig(rootfs string, registryIP net.IP) *Config {
	var config Config
	var defaultConfig = &Config{
		IP:     registryIP,
		Domain: SeaHub,
		Port:   "5000",
	}
	registryConfigPath := filepath.Join(rootfs, common.EtcDir, ConfigFile)
	if !osi.IsFileExist(registryConfigPath) {
		logrus.Debugf("default registry configuration is used: \n %+v", defaultConfig)
		return defaultConfig
	}
	err := yaml.UnmarshalFile(registryConfigPath, &config)
	if err != nil {
		logrus.Errorf("failed to read registry config: %v", err)
		return defaultConfig
	}
	if config.IP == nil {
		config.IP = defaultConfig.IP
	}
	if config.Port == "" {
		config.Port = defaultConfig.Port
	}
	if config.Domain == "" {
		config.Domain = defaultConfig.Domain
	}
	logrus.Debugf("The ultimate registry configration is: \n %+v", config)
	return &config
}
