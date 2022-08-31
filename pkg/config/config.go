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
	"github.com/imdario/mergo"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	k8sv1 "k8s.io/api/core/v1"
	k8sYaml "sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils/os"
)

const (
	Merge     = "merge"
	Overwrite = "overwrite"
)

type Interface interface {
	// Dump Config in Clusterfile to the cluster rootfs disk
	Dump(configs []v1.Config) error
}

type Dumper struct {
	Configs []v1.Config
	Cluster *v2.Cluster
}

func NewConfiguration(cluster *v2.Cluster) Interface {
	return &Dumper{
		Cluster: cluster,
	}
}

func (c *Dumper) Dump(configs []v1.Config) error {
	if configs == nil {
		logrus.Debug("clusterfile config is empty!")
		return nil
	}
	c.Configs = configs
	if err := c.WriteFiles(); err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles() error {
	if c.Configs == nil {
		logrus.Debug("empty config found")
		return nil
	}
	for _, config := range c.Configs {
		configData := []byte(config.Spec.Data)
		mountRoot := filepath.Join(common.DefaultClusterRootfsDir, c.Cluster.Name, "mount")
		mountDirs, err := ioutil.ReadDir(mountRoot)
		if err != nil {
			return err
		}
		convertSecret := strings.Contains(config.Spec.Process, toSecretProcessorName)
		for _, f := range mountDirs {
			if !f.IsDir() {
				continue
			}
			configPath := filepath.Join(mountRoot, f.Name(), config.Spec.Path)
			if convertSecret {
				if configData, err = convertSecretYaml(config, configPath); err != nil {
					return fmt.Errorf("faild to convert to secret file: %v", err)
				}
			}

			//Only files in yaml format are supported.
			//if Strategy is "Merge" will deeply merge each yaml file section.
			//if not, overwrite the whole file content with config data.
			if os.IsFileExist(configPath) && !convertSecret && config.Spec.Strategy == Merge {
				if configData, err = getMergeConfigData(configPath, configData); err != nil {
					return err
				}
			}
			err = os.NewCommonWriter(configPath).WriteFile(configData)
			if err != nil {
				return fmt.Errorf("failed to write config file %s: %v", configPath, err)
			}
		}
	}

	return nil
}

//getMergeConfigData merge data to each section of given file with overriding.
// given file is must be yaml marshalled.
func getMergeConfigData(filePath string, data []byte) ([]byte, error) {
	var (
		configs    [][]byte
		srcDataMap = make(map[string]interface{})
	)

	err := yaml.Unmarshal(data, &srcDataMap)
	if err != nil {
		return nil, fmt.Errorf("failed to load config data: %v", err)
	}

	contents, err := ioutil.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return nil, err
	}

	for _, rawCfgData := range bytes.Split(contents, []byte("---\n")) {
		destConfigMap := make(map[string]interface{})

		err = yaml.Unmarshal(rawCfgData, &destConfigMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config data from %s: %v", filePath, err)
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

func convertSecretYaml(config v1.Config, configPath string) ([]byte, error) {
	secret := k8sv1.Secret{}
	dataMap := make(map[string]string)
	if err := k8sYaml.Unmarshal([]byte(config.Spec.Data), &dataMap); err != nil {
		return nil, err
	}
	if os.IsFileExist(configPath) {
		rawData, err := ioutil.ReadFile(filepath.Clean(configPath))
		if err != nil {
			return nil, err
		}
		if err = k8sYaml.Unmarshal(rawData, &secret); err != nil {
			return nil, err
		}
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
