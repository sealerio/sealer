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
	"os"
	"path/filepath"
	"plugin"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
)

type Plugins interface {
	Dump(clusterfile string) error
	Load() error
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

// Load loads plugin configs in rootfs/plugin dir
func (c *PluginsProcessor) Load() error {
	c.Plugins = nil
	path := common.DefaultTheClusterRootfsPluginDir(c.ClusterName)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin dir %v", err)
	}
	for _, f := range files {
		if !utils.YamlMatcher(f.Name()) {
			continue
		}
		plugins, err := utils.DecodePlugins(filepath.Join(path, f.Name()))
		if err != nil {
			return fmt.Errorf("failed to load plugin %v", err)
		}
		c.Plugins = append(c.Plugins, plugins...)
	}
	return nil
}

func (c *PluginsProcessor) Run(cluster *v1.Cluster, phase Phase) error {
	var p Interface

	for _, config := range c.Plugins {
		if ext := filepath.Ext(config.Name); ext == ".so" {
			// load .so file from rootfs/plugin,if .so file not found,maybe not in the right phase.
			soFile := filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.ClusterName), config.Name)
			if !utils.IsExist(soFile) {
				return nil
			}
			out, err := c.loadOutOfTree(soFile)
			if err != nil {
				return err
			}
			return out.Run(Context{Cluster: cluster, Plugin: &config}, phase)
		}

		switch config.Spec.Type {
		case LabelPlugin:
			p = NewLabelsPlugin()
		case ShellPlugin:
			p = NewShellPlugin()
		case EtcdPlugin:
			p = NewEtcdBackupPlugin()
		case HostNamePlugin:
			p = NewHostnamePlugin()
		case ClusterCheckPlugin:
			p = NewClusterCheckerPlugin()
		default:
			return fmt.Errorf("not find plugin %v", config)
		}
		err := p.Run(Context{Cluster: cluster, Plugin: &config}, phase)
		if err != nil {
			return fmt.Errorf("failed to run plugin, %v", err)
		}
	}
	return nil
}

func (c *PluginsProcessor) loadOutOfTree(soFile string) (Interface, error) {
	plug, err := plugin.Open(soFile)
	if err != nil {
		return nil, err
	}
	//look up the exposed variable named `Plugin`
	symbol, err := plug.Lookup(Plugin)
	if err != nil {
		return nil, err
	}

	p, ok := symbol.(Interface)
	if !ok {
		return nil, fmt.Errorf("failed to find GOLANG plugin symbol")
	}
	return p, nil
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
		var name = config.Name
		if !utils.YamlMatcher(name) {
			name = fmt.Sprintf("%s.yaml", name)
		}

		err := utils.MarshalYamlToFile(filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.ClusterName), name), config)
		if err != nil {
			return fmt.Errorf("write plugin metadata fileed %v", err)
		}
	}

	return nil
}
