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

	"github.com/alibaba/sealer/logger"

	"github.com/alibaba/sealer/utils"
)

type RegistryConfig struct {
	IP     string `yaml:"ip,omitempty"`
	Domain string `yaml:"domain,omitempty"`
}

func (d *Default) getRegistryHost() (host string) {
	cf := d.getRegistryConfig()
	return fmt.Sprintf("%s %s", cf.IP, cf.Domain)
}

func (d *Default) getRegistryConfig() *RegistryConfig {
	var config RegistryConfig
	var Default = &RegistryConfig{
		IP:     d.Masters[0],
		Domain: SeaHub,
	}
	registryConfigPath := filepath.Join(d.Rootfs, "/etc/registry.yaml")
	if !utils.IsFileExist(registryConfigPath) {
		return Default
	}
	err := utils.UnmarshalYamlFile(registryConfigPath, &config)
	if err != nil {
		logger.Error("Failed to read registry config! ")
		return Default
	}
	if config.IP == "" {
		config.IP = d.Masters[0]
	}
	if config.Domain == "" {
		config.Domain = SeaHub
	}
	return &config
}

const registryName = "sealer-registry"

//Only use this for join and init, due to the initiation operations
func (d *Default) EnsureRegistry() error {
	cf := d.getRegistryConfig()
	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", d.Rootfs, d.Rootfs)
	return d.SSH.CmdAsync(cf.IP, cmd)
}

func (d *Default) RecycleRegistry() error {
	cf := d.getRegistryConfig()
	cmd := fmt.Sprintf("docker stop %s || true && docker rm %s || true", registryName, registryName)
	return d.SSH.CmdAsync(cf.IP, cmd)
}
