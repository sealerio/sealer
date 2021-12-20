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
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/logger"
	v1 "github.com/alibaba/sealer/types/api/v1"
	"github.com/alibaba/sealer/utils"
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
	// Dump Config in Clusterfile to the cluster rootfs disk
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
	configs, err := utils.DecodeConfigs(clusterfile)
	if err != nil {
		return fmt.Errorf("failed to dump config %v", err)
	}
	c.Configs = configs
	err = c.WriteFiles()
	if err != nil {
		return fmt.Errorf("failed to write config files %v", err)
	}
	return nil
}

func (c *Dumper) WriteFiles() error {
	if c.Configs == nil {
		logger.Debug("config is nil")
		return nil
	}
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
