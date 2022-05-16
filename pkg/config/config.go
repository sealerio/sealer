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
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/sealerio/sealer/utils/os"

	"gopkg.in/yaml.v3"
	k8sv1 "k8s.io/api/core/v1"
	k8sYaml "sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

/*
config in Clusterfile:

apiVersion: sealer.aliyun.com/v1alpha1
kind: Config
metadata:
  name: redis-config
spec:
  path: etc/redis-config.yaml
  data: |
       redis-user: root
       redis-passwd: xxx

Dump will dump the config to etc/redis-config.yaml file
*/

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
		logger.Debug("clusterfile config is empty!")
		return nil
	}
	c.Configs = configs
	if err := c.WriteFiles(); err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles() (err error) {
	if c.Configs == nil {
		logger.Debug("empty config found")
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
			//only the YAML format is supported
			if os.IsFileExist(configPath) && !convertSecret && config.Spec.Strategy == Merge {
				if configData, err = getMergeConfigData(configPath, configData); err != nil {
					return err
				}
			}
			err = os.NewCommonWriter(configPath).WriteFile(configData)
			if err != nil {
				return fmt.Errorf("write config file failed %v", err)
			}
		}
	}

	return nil
}

//merge the contents of data into the path file
func getMergeConfigData(path string, data []byte) ([]byte, error) {
	var configs [][]byte
	context, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	mergeConfigMap := make(map[string]interface{})
	err = yaml.Unmarshal(data, &mergeConfigMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal merge map: %v", err)
	}
	for _, rawCfgData := range bytes.Split(context, []byte("---\n")) {
		configMap := make(map[string]interface{})
		err = yaml.Unmarshal(rawCfgData, &configMap)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %v", err)
		}
		if len(configMap) == 0 {
			continue
		}
		deepMerge(&configMap, &mergeConfigMap)
		cfg, err := k8sYaml.Marshal(&configMap)
		if err != nil {
			return nil, err
		}
		configs = append(configs, cfg)
	}
	return bytes.Join(configs, []byte("\n---\n")), nil
}

func deepMerge(dst, src *map[string]interface{}) {
	for srcK, srcV := range *src {
		dstV, ok := (*dst)[srcK]
		if !ok {
			continue
		}
		dV, ok := dstV.(map[string]interface{})
		// dstV is string type
		if !ok {
			(*dst)[srcK] = srcV
			continue
		}
		sV, ok := srcV.(map[string]interface{})
		if !ok {
			continue
		}
		deepMerge(&dV, &sV)
		(*dst)[srcK] = dV
	}
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
