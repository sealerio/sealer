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

package clusterfile

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/pkg/config"
	"github.com/sealerio/sealer/pkg/env"
	"github.com/sealerio/sealer/pkg/runtime/kubernetes"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	yaml2 "github.com/sealerio/sealer/utils"

	"github.com/sirupsen/logrus"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type PreProcessor interface {
	Process() error
}

func NewPreProcessor(path string) PreProcessor {
	return &ClusterFile{path: path}
}

func (c *ClusterFile) GetPipeLine() ([]func() error, error) {
	var todoList []func() error
	todoList = append(todoList,
		c.PrePareCluster,
		c.PrePareEnv,
		c.PrePareConfigs,
	)
	return todoList, nil
}

func (c *ClusterFile) Process() error {
	pipeLine, err := c.GetPipeLine()
	if err != nil {
		return err
	}
	for _, f := range pipeLine {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterFile) PrePareEnv() error {
	clusterFileData, err := env.NewEnvProcessor(&c.Cluster).Process(c.path)
	if err != nil {
		return err
	}

	if err = c.DecodePlugins(clusterFileData); err != nil {
		return err
	}

	if err = c.DecodeConfigs(clusterFileData); err != nil {
		return err
	}

	return c.DecodeKubeadmConfig(clusterFileData)
}

func (c *ClusterFile) PrePareConfigs() error {
	var configs []v1.Config
	for _, c := range c.GetConfigs() {
		cfg := c
		err := config.NewProcessorsAndRun(&cfg)
		if err != nil {
			return err
		}
		configs = append(configs, cfg)
	}
	c.Configs = configs
	return nil
}

func (c *ClusterFile) PrePareCluster() error {
	f, err := os.Open(c.path)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil {
			logrus.Warnf("failed to close file")
		}
	}()
	d := yaml.NewYAMLOrJSONDecoder(f, 4096)
	for {
		ext := runtime2.RawExtension{}
		if err = d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			continue
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		err = c.DecodeCluster(ext.Raw)
		if err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to decode cluster from %s, %v", c.path, err)
}

func (c *ClusterFile) DecodeCluster(data []byte) error {
	cluster, err := yaml2.DecodeCRDFromByte(data, common.Cluster)
	if err != nil {
		return err
	}
	c.Cluster = *cluster.(*v2.Cluster)
	return nil
}

func (c *ClusterFile) DecodeConfigs(data []byte) error {
	configs, err := yaml2.DecodeCRDFromByte(data, common.Config)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	cfg := configs.([]v1.Config)
	c.Configs = cfg
	return nil
}

func (c *ClusterFile) DecodePlugins(data []byte) error {
	plugs, err := yaml2.DecodeCRDFromByte(data, common.Plugin)
	if err != nil {
		if err == io.EOF {
			return nil
		}
		return err
	}
	plugins := plugs.([]v1.Plugin)
	c.Plugins = plugins
	return nil
}

func (c *ClusterFile) DecodeKubeadmConfig(data []byte) error {
	kubeadmConfig, err := kubernetes.LoadKubeadmConfigs(string(data), yaml2.DecodeCRDFromString)
	if err != nil {
		return err
	}
	c.KubeConfig = kubeadmConfig
	return nil
}
