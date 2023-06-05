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

package config

import (
	"bytes"
	"fmt"
	stdos "os"
	"path/filepath"
	"strings"

	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	k8sv1 "k8s.io/api/core/v1"
	k8sYaml "sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/pkg/rootfs"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/os"
)

const (
	Merge = "merge"
)

type Interface interface {
	// Dump Configs from Clusterfile to the cluster rootfs
	Dump(configs []v1.Config) error
}

type Dumper struct {
	// rootPath typically is cluster image mounted base directory.
	rootPath string
}

func NewConfiguration(rootPath string) Interface {
	return &Dumper{
		rootPath: rootPath,
	}
}

func (c *Dumper) Dump(configs []v1.Config) error {
	if len(configs) == 0 {
		logrus.Debug("no config is found")
		return nil
	}

	if err := c.WriteFiles(configs); err != nil {
		return fmt.Errorf("failed to dump config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles(configs []v1.Config) error {
	for _, config := range configs {
		//#nosec
		if err := NewProcessorsAndRun(&config); err != nil {
			return err
		}

		configData := []byte(config.Spec.Data)
		path := config.Spec.Path
		if config.Spec.APPName != "" {
			path = filepath.Join(rootfs.GlobalManager.App().Root(), config.Spec.APPName, path)
		}
		configPath := filepath.Join(c.rootPath, path)

		logrus.Debugf("dumping config:%+v\n on the target file", config)
		if !os.IsFileExist(configPath) {
			err := os.NewCommonWriter(configPath).WriteFile(configData)
			if err != nil {
				return fmt.Errorf("failed to overwrite config file %s: %v", configPath, err)
			}
			continue
		}

		contents, err := stdos.ReadFile(filepath.Clean(configPath))
		if err != nil {
			return err
		}

		// todo: its strange to use config.Spec.Process to control config dump strategy.
		if strings.Contains(config.Spec.Process, toSecretProcessorName) {
			if configData, err = convertSecretYaml(contents, configData); err != nil {
				return fmt.Errorf("faild to convert to secret file: %v", err)
			}
		}
		//Only files in yaml format are supported.
		//if Strategy is "Merge" will deeply merge each yaml file section.
		//if not, overwrite the whole file content with config data.
		if config.Spec.Strategy == Merge {
			if configData, err = getMergeConfigData(contents, configData); err != nil {
				return err
			}
		}

		err = os.NewCommonWriter(configPath).WriteFile(configData)
		if err != nil {
			return fmt.Errorf("failed to write config file %s: %v", configPath, err)
		}
	}
	return nil
}

// getMergeConfigData merge data to each section of given file with overriding.
// given file is must be yaml marshalled.
func getMergeConfigData(contents, data []byte) ([]byte, error) {
	var (
		configs    [][]byte
		srcDataMap = make(map[string]interface{})
	)

	err := yaml.Unmarshal(data, &srcDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to load config data: %v", err)
	}

	for _, rawCfgData := range bytes.Split(contents, []byte("---\n")) {
		destConfigMap := make(map[string]interface{})

		err = yaml.Unmarshal(rawCfgData, &destConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config data: %v", err)
		}

		err = mergo.Merge(&destConfigMap, &srcDataMap, mergo.WithOverride)
		if err != nil {
			return nil, fmt.Errorf("failed to merge config: %v", err)
		}

		cfg, err := yaml.Marshal(destConfigMap)
		if err != nil {
			return nil, err
		}

		configs = append(configs, cfg)
	}
	return bytes.Join(configs, []byte("---\n")), nil
}

func convertSecretYaml(contents, data []byte) ([]byte, error) {
	secret := k8sv1.Secret{}
	dataMap := make(map[string]string)
	if err := k8sYaml.Unmarshal(data, &dataMap); err != nil {
		return nil, err
	}

	if err := k8sYaml.Unmarshal(contents, &secret); err != nil {
		return nil, err
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	//set secret data
	for k, v := range dataMap {
		v := []byte(v)
		secret.Data[k] = v
	}
	return k8sYaml.Marshal(secret)
}
