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

	"github.com/alibaba/sealer/logger"
	runtime2 "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/config"
	"github.com/alibaba/sealer/pkg/env"
	"github.com/alibaba/sealer/pkg/runtime"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
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
	err = c.DecodePlugins(clusterFileData)
	if err != nil && err != ErrTypeNotFound {
		return err
	}
	err = c.DecodeConfigs(clusterFileData)
	if err != nil && err != ErrTypeNotFound {
		return err
	}
	err = c.DecodeKubeadmConfig(clusterFileData)
	if err != nil && err != ErrTypeNotFound {
		return err
	}
	return nil
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
			logger.Fatal("failed to close file")
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
	cluster, err := GetClusterFromDataCompatV1(data)
	if err != nil {
		return err
	}
	c.Cluster = *cluster
	return nil
}

func (c *ClusterFile) DecodeConfigs(data []byte) error {
	configs, err := utils.DecodeV1CRDFromReader(bytes.NewReader(data), common.Config)
	if err != nil {
		return err
	}
	if configs == nil {
		return ErrTypeNotFound
	}
	cfgs := configs.([]v1.Config)
	c.Configs = cfgs
	return nil
}

func (c *ClusterFile) DecodePlugins(data []byte) error {
	plugs, err := utils.DecodeV1CRDFromReader(bytes.NewReader(data), common.Plugin)
	if err != nil {
		return err
	}
	if plugs == nil {
		return ErrTypeNotFound
	}
	plugins := plugs.([]v1.Plugin)
	c.Plugins = plugins
	return nil
}

func (c *ClusterFile) DecodeKubeadmConfig(data []byte) error {
	kubeadmConfig, err := runtime.LoadKubeadmConfigs(string(data), runtime.DecodeCRDFromString)
	if err != nil {
		return err
	}
	if kubeadmConfig == nil {
		return ErrTypeNotFound
	}
	c.KubeConfig = kubeadmConfig
	return nil
}
