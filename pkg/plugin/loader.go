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

package plugin

import (
	"bufio"
	"bytes"
	"fmt"
	v1 "github.com/sealerio/sealer/types/api/v1"
	"github.com/sealerio/sealer/utils/yaml"
	"io"
	"io/ioutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"os"
	"path/filepath"
	"strings"
)

// LoadFromFile load plugin config files from $rootfs/plugins dir.
func LoadFromFile(pluginPath string) ([]v1.Plugin, error) {
	_, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		return nil, nil
	}

	files, err := ioutil.ReadDir(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to ReadDir plugin dir %s: %v", pluginPath, err)
	}

	var plugins []v1.Plugin
	for _, f := range files {
		if !yaml.Matcher(f.Name()) {
			continue
		}

		pluginFile := filepath.Join(pluginPath, f.Name())
		pluginList, err := decodePluginFile(pluginFile)
		if err != nil {
			return nil, fmt.Errorf("failed to decode plugin file %s: %v", pluginFile, err)
		}

		var plugs []v1.Plugin
		for _, p := range pluginList {
			for _, cp := range plugins {
				if !isSamePluginSpec(p, cp) {
					plugs = append(plugs, p)
				}
			}
		}

		plugins = append(plugins, plugs...)
	}

	return plugins, nil
}

func decodePluginFile(pluginFile string) ([]v1.Plugin, error) {
	var plugins []v1.Plugin
	data, err := ioutil.ReadFile(filepath.Clean(pluginFile))
	if err != nil {
		return nil, err
	}

	decoder := k8syaml.NewYAMLToJSONDecoder(bufio.NewReaderSize(bytes.NewReader(data), 4096))
	for {
		ext := runtime.RawExtension{}
		if err := decoder.Decode(&ext); err != nil {
			if err == io.EOF {
				return plugins, nil
			}
			return nil, err
		}

		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}
		metaType := metav1.TypeMeta{}
		if err := k8syaml.Unmarshal(ext.Raw, &metaType); err != nil {
			return nil, fmt.Errorf("failed to decode TypeMeta: %v", err)
		}

		var plu v1.Plugin
		if err := k8syaml.Unmarshal(ext.Raw, &plu); err != nil {
			return nil, fmt.Errorf("failed to decode %s[%s]: %v", metaType.Kind, metaType.APIVersion, err)
		}

		plu.Spec.Data = strings.TrimSuffix(plu.Spec.Data, "\n")
		plugins = append(plugins, plu)
	}
}
