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
	"github.com/alibaba/sealer/utils/mount"
)

const (
	RegistryName       = "sealer-registry"
	RegistryBindDest   = "/var/lib/registry"
	RegistryMountUpper = "/var/lib/sealer/tmp/upper"
	RegistryMountWork  = "/var/lib/sealer/tmp/work"
)

type RegistryConfig struct {
	IP     string `yaml:"ip,omitempty"`
	Domain string `yaml:"domain,omitempty"`
	Port   string `yaml:"port,omitempty"`
}

func getRegistryHost(rootfs, defaultRegistry string) (host string) {
	cf := GetRegistryConfig(rootfs, defaultRegistry)
	ip, _ := utils.GetSSHHostIPAndPort(cf.IP)
	return fmt.Sprintf("%s %s", ip, cf.Domain)
}

func GetRegistryConfig(rootfs, defaultRegistry string) *RegistryConfig {
	var config RegistryConfig
	var DefaultConfig = &RegistryConfig{
		IP:     defaultRegistry,
		Domain: SeaHub,
		Port:   "5000",
	}
	registryConfigPath := filepath.Join(rootfs, "/etc/registry.yaml")
	if !utils.IsFileExist(registryConfigPath) {
		logger.Debug("use default registry config")
		return DefaultConfig
	}
	err := utils.UnmarshalYamlFile(registryConfigPath, &config)
	logger.Info(fmt.Sprintf("show registry info, IP: %s, Domain: %s", config.IP, config.Domain))
	if err != nil {
		logger.Error("Failed to read registry config! ")
		return DefaultConfig
	}
	if config.IP == "" {
		config.IP = DefaultConfig.IP
	} else {
		ip, port := utils.GetSSHHostIPAndPort(config.IP)
		config.IP = fmt.Sprintf("%s:%s", ip, port)
	}
	if config.Port == "" {
		config.Port = DefaultConfig.Port
	}
	if config.Domain == "" {
		config.Domain = DefaultConfig.Domain
	}
	return &config
}

//Only use this for join and init, due to the initiation operations
func (d *Default) EnsureRegistry() error {
	cf := GetRegistryConfig(d.Rootfs, d.Masters[0])
	mkdir := fmt.Sprintf("rm -rf %s %s && mkdir -p %s %s", RegistryMountUpper, RegistryMountWork,
		RegistryMountUpper, RegistryMountWork)

	mountCmd := fmt.Sprintf("%s && mount -t overlay overlay -o lowerdir=%s,upperdir=%s,workdir=%s %s", mkdir,
		d.Rootfs,
		RegistryMountUpper, RegistryMountWork, d.Rootfs)
	isMount, _ := mount.GetRemoteMountDetails(d.SSH, cf.IP, d.Rootfs)
	if isMount {
		mountCmd = fmt.Sprintf("umount %s && %s", d.Rootfs, mountCmd)
	}
	if err := d.SSH.CmdAsync(cf.IP, mountCmd); err != nil {
		return err
	}

	cmd := fmt.Sprintf("cd %s/scripts && sh init-registry.sh %s %s", d.Rootfs, cf.Port, fmt.Sprintf("%s/registry", d.Rootfs))
	return d.SSH.CmdAsync(cf.IP, cmd)
}

func (d *Default) RecycleRegistry() error {
	cf := GetRegistryConfig(d.Rootfs, d.Masters[0])
	delDir := fmt.Sprintf("rm -rf %s %s", RegistryMountUpper, RegistryMountWork)
	isMount, _ := mount.GetRemoteMountDetails(d.SSH, cf.IP, d.Rootfs)
	if isMount {
		delDir = fmt.Sprintf("umount %s && %s", d.Rootfs, delDir)
	}
	cmd := fmt.Sprintf("if docker inspect %s;then docker rm -f %s;fi && %s ", RegistryName, RegistryName, delDir)
	return d.SSH.CmdAsync(cf.IP, cmd)
}
