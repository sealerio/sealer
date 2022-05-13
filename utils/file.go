// Copyright © 2022 Alibaba Group Holding Ltd.
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

package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func DecodeCluster(filepath string) (clusters []v1.Cluster, err error) {
	decodeClusters, err := DecodeV1CRD(filepath, common.Cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cluster from %s, %v", filepath, err)
	}
	clusters = decodeClusters.([]v1.Cluster)
	return
}

func DecodeConfigs(filepath string) (configs []v1.Config, err error) {
	decodeConfigs, err := DecodeV1CRD(filepath, common.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config from %s, %v", filepath, err)
	}
	configs = decodeConfigs.([]v1.Config)
	return
}

func DecodePlugins(filepath string) (plugins []v1.Plugin, err error) {
	decodePlugins, err := DecodeV1CRD(filepath, common.Plugin)
	if err != nil {
		return nil, fmt.Errorf("failed to decode plugin from %s, %v", filepath, err)
	}
	plugins = decodePlugins.([]v1.Plugin)
	return
}

func DecodeV1CRD(filepath string, kind string) (out interface{}, err error) {
	file, err := os.Open(path.Clean(filepath))
	if err != nil {
		return nil, fmt.Errorf("failed to dump config %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			logger.Warn("failed to dump config close clusterfile failed %v", err)
		}
	}()
	return DecodeV1CRDFromReader(file, kind)
}

func DecodeV1CRDFromReader(reader io.Reader, kind string) (out interface{}, err error) {
	var (
		i        interface{}
		clusters []v1.Cluster
		configs  []v1.Config
		plugins  []v1.Plugin
	)

	d := yaml.NewYAMLOrJSONDecoder(reader, 4096)

	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		// TODO: This needs to be able to handle object in other encodings and schemas.
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		// ext.Raw
		switch kind {
		case common.Cluster:
			cluster := v1.Cluster{}
			err := yaml.Unmarshal(ext.Raw, &cluster)
			if err != nil {
				return nil, fmt.Errorf("decode cluster failed %v", err)
			}
			if cluster.Kind == common.Cluster {
				clusters = append(clusters, cluster)
			}
			i = clusters
		case common.Config:
			config := v1.Config{}
			err := yaml.Unmarshal(ext.Raw, &config)
			if err != nil {
				return nil, fmt.Errorf("decode config failed %v", err)
			}
			if config.Kind == common.Config {
				configs = append(configs, config)
			}
			i = configs
		case common.Plugin:
			plugin := v1.Plugin{}
			err := yaml.Unmarshal(ext.Raw, &plugin)
			if err != nil {
				return nil, fmt.Errorf("decode plugin failed %v", err)
			}
			if plugin.Kind == common.Plugin {
				plugins = append(plugins, plugin)
			}
			i = plugins
		}
	}

	return i, nil
}
