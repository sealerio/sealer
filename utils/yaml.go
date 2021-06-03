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
	"io/ioutil"

	"sigs.k8s.io/yaml"
)

func UnmarshalYamlFile(file string, obj interface{}) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, obj)
	return err
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
