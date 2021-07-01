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

	"github.com/alibaba/sealer/common"
)

func (d *Default) getRegistryHost() (host string) {
	ip, domain := d.getRegistryConfig()
	return fmt.Sprintf("%s %s", ip, domain)
}

func (d *Default) getRegistryConfig() (host, domain string) {
	var config map[string]string
	registryConfigPath := filepath.Join(common.DefaultClusterBaseDir(d.ClusterName), "etc/registry.yaml")
	if utils.IsFileExist(registryConfigPath) {
		err := utils.UnmarshalYamlFile(registryConfigPath, &config)
		if err == nil {
			return config["ip"], config["domain"]
		}
		logger.Error("Failed to read registry config! ")
	}
	return d.Masters[0], SeaHub
}

const registryName = "sealer-registry"

//Only use this for join and init, due to the initiation operations
func (d *Default) EnsureRegistry() error {
	ip, _ := d.getRegistryConfig()
	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh 5000 %s/registry", d.Rootfs, d.Rootfs)
	return d.SSH.CmdAsync(ip, cmd)
}

func (d *Default) RecycleRegistry() error {
	ip, _ := d.getRegistryConfig()
	cmd := fmt.Sprintf("docker stop %s || true && docker rm %s || true", registryName, registryName)
	return d.SSH.CmdAsync(ip, cmd)
}
