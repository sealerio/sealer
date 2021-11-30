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
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/alibaba/sealer/common"
	v1 "github.com/alibaba/sealer/types/api/v1"
)

func UnmarshalYamlFile(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file %s to %s", file, reflect.TypeOf(obj))
	}
	return nil
}

func MarshalYamlToFile(file string, obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	if err = WriteFile(file, data); err != nil {
		return err
	}
	return nil
}

func SaveClusterfile(cluster *v1.Cluster) error {
	fileName := common.GetClusterWorkClusterfile(cluster.Name)
	err := MkFileFullPathDir(fileName)
	if err != nil {
		return fmt.Errorf("mkdir failed %s %v", fileName, err)
	}
	err = MarshalYamlToFile(fileName, cluster)
	if err != nil {
		return fmt.Errorf("marshal cluster file failed %v", err)
	}
	return nil
}

func YamlMatcher(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
