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

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func UnmarshalYamlFile(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file %s to %s: %v", file, reflect.TypeOf(obj), err)
	}
	return nil
}

func MarshalYamlToFile(file string, obj interface{}) error {
	switch cluster := obj.(type) {
	case *v1.Cluster:
		if cluster.Spec.SSH.Encrypted {
			break
		}
		passwd, err := AesEncrypt([]byte(cluster.Spec.SSH.Passwd))
		if err != nil {
			return err
		}
		cluster.Spec.SSH.Passwd = passwd
		cluster.Spec.SSH.Encrypted = true
	case *v2.Cluster:
		if cluster.Spec.SSH.Encrypted {
			break
		}
		passwd, err := AesEncrypt([]byte(cluster.Spec.SSH.Passwd))
		if err != nil {
			return err
		}
		cluster.Spec.SSH.Passwd = passwd
		cluster.Spec.SSH.Encrypted = true
	default:
	}
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	if err = WriteFile(file, data); err != nil {
		return err
	}
	return nil
}

func SaveClusterInfoToFile(cluster runtime.Object, clusterName string) error {
	fileName := common.GetClusterWorkClusterfile(clusterName)
	err := MkFileFullPathDir(fileName)
	if err != nil {
		return fmt.Errorf("mkdir failed %s %v", fileName, err)
	}
	cluster = cluster.DeepCopyObject()
	err = MarshalYamlToFile(fileName, cluster)
	if err != nil {
		return fmt.Errorf("marshal cluster file failed %v", err)
	}
	return nil
}

func MarshalYamlConfigs(configs ...interface{}) ([]byte, error) {
	var cfgs [][]byte
	for _, cfg := range configs {
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return nil, err
		}
		cfgs = append(cfgs, data)
	}
	return bytes.Join(cfgs, []byte("\n---\n")), nil
}

func YamlMatcher(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
