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

package plugin

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

/*
config in PluginConfig:

apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=master
  data: |
    kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule

Dump will dump the config to etc/redis-config.yaml file
*/

type Plugins interface {
	Dump(clusterfile string) error
	Run(cluster *v1.Cluster, phase Phase) error
}

type PluginsProcessor struct {
	Plugins     []v1.Plugin
	ClusterName string
}

func NewPlugins(clusterName string) Plugins {
	return &PluginsProcessor{
		ClusterName: clusterName,
		Plugins:     []v1.Plugin{},
	}
}

// load plugin configs in rootfs/plugin dir
func (c *PluginsProcessor) load() error {
	c.Plugins = nil
	files, err := ioutil.ReadDir(common.DefaultTheClusterRootfsPluginDir(c.ClusterName))
	if err != nil {
		return fmt.Errorf("failed to load plugin dir %v", err)
	}
	for _, f := range files {
		plugin := v1.Plugin{}
		err := utils.UnmarshalYamlFile(f.Name(), plugin)
		if err != nil {
			return fmt.Errorf("failed to load plugin %v", err)
		}
		c.Plugins = append(c.Plugins, plugin)
	}

	return nil
}

func (c *PluginsProcessor) Run(cluster *v1.Cluster, phase Phase) error {
	var p Interface

	err := c.load()
	if err != nil {
		return err
	}
	for _, config := range c.Plugins {
		switch config.Spec.Type {
		case LabelPlugin:
			p = NewLabelsPlugin()
		case ShellPlugin:
			p = NewShellPlugin()
		case EtcdPlugin:
			p = NewEtcdBackupPlugin()
		case HostNamePlugin:
			p = NewHostnamePlugin()
		default:
			return fmt.Errorf("not find plugin %s", config.Name)
		}
		err := p.Run(Context{Cluster: cluster, Plugin: &config}, phase)
		if err != nil {
			return fmt.Errorf("failed to run plugin, %v", err)
		}
	}
	return nil
}

func (c *PluginsProcessor) Dump(clusterfile string) error {
	if clusterfile == "" {
		logger.Debug("clusterfile is empty!")
		return nil
	}
	plugins, err := utils.DecodePlugins(clusterfile)
	if err != nil {
		return err
	}
	c.Plugins = plugins
	err = c.WriteFiles()
	if err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *PluginsProcessor) WriteFiles() error {
	if len(c.Plugins) == 0 {
		logger.Debug("plugins is nil")
		return nil
	}
	for _, config := range c.Plugins {
		err := utils.WriteFile(filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.ClusterName), config.ObjectMeta.Name), []byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("write config fileed %v", err)
		}
	}

	return nil
}
