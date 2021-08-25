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
	"io"
	"io/ioutil"
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

type Interface interface {
	// dump Config in Clusterfile to the cluster rootfs disk
	Dump(clusterfile string) error
}

type Dumper struct {
	Configs     []v1.Config
	ClusterName string
}

func NewConfiguration(clusterName string) Interface {
	return &Dumper{
		ClusterName: clusterName,
	}
}

func (c *Dumper) Dump(clusterfile string) error {
	if clusterfile == "" {
		logger.Debug("clusterfile is empty!")
		return nil
	}
	err := c.DecodeConfig(clusterfile)
	if err != nil {
		return fmt.Errorf("failed to dump config %v", err)
	}

	err = c.WriteFiles()
	if err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles() error {
	for _, config := range c.Configs {
		err := utils.WriteFile(filepath.Join(common.DefaultTheClusterRootfsDir(c.ClusterName), config.Spec.Path), []byte(config.Spec.Data))
		if err != nil {
			return fmt.Errorf("write config fileed %v", err)
		}
		err = ioutil.WriteFile(filepath.Join(common.DefaultMountCloudImageDir(c.ClusterName), config.Spec.Path), []byte(config.Spec.Data), common.FileMode0644)
		if err != nil {
			return fmt.Errorf("write config file failed %v", err)
		}
	}

	return nil
}

func (c *Dumper) DecodeConfig(clusterfile string) error {
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
		err := c.decodeConfig(ext.Raw)
		if err != nil {
			return fmt.Errorf("failed to decode config file %v", err)
		}
	}
	return nil
}

func (c *Dumper) decodeConfig(Body []byte) error {
	config := v1.Config{}
	err := yaml.Unmarshal(Body, &config)
	if err != nil {
		return fmt.Errorf("decode config failed %v", err)
	}
	if config.Kind == common.CRDConfig {
		c.Configs = append(c.Configs, config)
	}

	return nil
}
