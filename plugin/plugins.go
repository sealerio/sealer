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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
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

type PluginsProcesser struct {
	plugins     []v1.Plugin
	clusterName string
}

func NewPlugins(clusterName string) Plugins {
	return &PluginsProcesser{
		clusterName: clusterName,
		plugins:     []v1.Plugin{},
	}
}

func (c *PluginsProcesser) Run(cluster *v1.Cluster, phase Phase) error {
	for _, config := range c.plugins {
		switch config.Name {
		case "LABEL":
			l := LabelsNodes{}
			err := l.Run(Context{Cluster: cluster, Plugin: &config}, phase)
			if err != nil {
				return err
			}
		case "SHELL":
			s := Sheller{}
			err := s.Run(Context{Cluster: cluster, Plugin: &config}, phase)
			if err != nil {
				return err
			}
		case "ETCD":
			e := EtcdBackupPlugin{}
			err := e.Run(Context{Cluster: cluster, Plugin: &config}, phase)
			if err != nil {
				return err
			}
		case "HOSTNAME":
			h := HostnamePlugin{}
			err := h.Run(Context{Cluster: cluster, Plugin: &config}, phase)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("not find plugin %s", config.Name)
		}
	}
	return nil
}

func (c *PluginsProcesser) Dump(clusterfile string) error {
	if clusterfile == "" {
		logger.Debug("clusterfile is empty!")
		return nil
	}
	file, err := os.Open(clusterfile)
	if err != nil {
		return fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()

	d := yaml.NewYAMLOrJSONDecoder(file, 4096)
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		// ext.Raw
		err := c.DecodeConfig(ext.Raw)
		if err != nil {
			return fmt.Errorf("failed to decode config file %v", err)
		}
	}

	err = c.WriteFiles()
	if err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *PluginsProcesser) WriteFiles() error {
	if len(c.plugins) == 0 {
		logger.Debug("config is nil")
		return nil
	}
	for _, config := range c.plugins {
		err := utils.WriteFile(filepath.Join(common.DefaultTheClusterRootfsPluginDir(c.clusterName), config.ObjectMeta.Name), []byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("write config fileed %v", err)
		}
	}

	return nil
}

func (c *PluginsProcesser) DecodeConfig(Body []byte) error {
	config := v1.Plugin{}
	err := yaml.Unmarshal(Body, &config)
	if err != nil {
		return fmt.Errorf("decode config failed %v", err)
	}
	if config.Kind == common.CRDPlugin {
		c.plugins = append(c.plugins, config)
	}
	return nil
}
