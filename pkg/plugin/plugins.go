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
	"strings"

	"github.com/sealerio/sealer/utils/slice"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
	"github.com/sealerio/sealer/utils/platform"
)

type InvalidPluginTypeError struct {
	Name string
}

func (err InvalidPluginTypeError) Error() string {
	return fmt.Sprintf("plugin type not registered: %s", err.Name)
}

type Plugins interface {
	Load() error
	Run(host []string, phase Phase) error
}

// PluginsProcessor : process two list: plugin config list and embed pluginFactories that contains plugin interface.
type PluginsProcessor struct {
	// plugin config list
	Plugins []v1.Plugin
	Cluster *v2.Cluster
}

//plugins form Clusterfile
func NewPlugins(cluster *v2.Cluster, plugins []v1.Plugin) Plugins {
	return &PluginsProcessor{
		Cluster: cluster,
		Plugins: plugins,
	}
}

// Load plugin configs and shared object(.so) file from $mountRootfs/plugins dir.
func (c *PluginsProcessor) Load() error {
	path := filepath.Join(platform.DefaultMountCloudImageDir(c.Cluster.Name), "plugins")
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return fmt.Errorf("failed to load plugin dir %v", err)
	}
	for _, f := range files {
		// load shared object(.so) file
		if filepath.Ext(f.Name()) == ".so" {
			soFile := filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.Cluster.Name), f.Name())
			p, pt, err := c.loadOutOfTree(soFile)
			if err != nil {
				return err
			}
			Register(pt, p)
		}
		if utils.YamlMatcher(f.Name()) {
			plugins, err := utils.DecodePlugins(filepath.Join(path, f.Name()))
			if err != nil {
				return fmt.Errorf("failed to load plugin %v", err)
			}
			c.Plugins = append(c.Plugins, plugins...)
		}
	}
	return nil
}

// Run execute each in-tree or out-of-tree plugin by traversing the plugin list.
func (c *PluginsProcessor) Run(host []string, phase Phase) error {
	for _, plugin := range c.Plugins {
		if slice.NotIn(string(phase), strings.Split(plugin.Spec.Action, "|")) {
			continue
		}
		p, ok := pluginFactories[plugin.Spec.Type]
		// if we use cluster file dump plugin config,some plugin load after mount rootfs,
		// we still need to return those not find error.
		// apply module to judged whether to show errors.
		if !ok {
			return InvalidPluginTypeError{plugin.Spec.Type}
		}
		// #nosec
		err := p.Run(Context{Cluster: c.Cluster, Host: host, Plugin: &plugin}, phase)
		if err != nil {
			return fmt.Errorf("failed to run plugin %s: %v", plugin.Name, err)
		}
	}
	return nil
}

func (c *PluginsProcessor) loadOutOfTree(soFile string) (Interface, string, error) {
	plug, err := plugin.Open(soFile)
	if err != nil {
		return nil, "", err
	}
	//look up the exposed variable named `Plugin`
	symbol, err := plug.Lookup(Plugin)
	if err != nil {
		return nil, "", err
	}

	p, ok := symbol.(Interface)
	if !ok {
		return nil, "", fmt.Errorf("failed to find Plugin symbol")
	}

	//look up the exposed variable named `PluginType`
	pts, err := plug.Lookup(PluginType)
	if err != nil {
		return nil, "", err
	}

	pt, ok := pts.(*string)
	if !ok {
		return nil, "", fmt.Errorf("failed to find PluginType symbol")
	}
	return p, *pt, nil
}
