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

package yaml

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	osi "github.com/sealerio/sealer/utils/os"

	"sigs.k8s.io/yaml"
)

func UnmarshalFile(file string, obj interface{}) error {
	data, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, obj)
	if err != nil {
		return fmt.Errorf("failed to unmarshal file %s to %s: %v", file, reflect.TypeOf(obj), err)
	}
	return nil
}

func MarshalToFile(file string, obj interface{}) error {
	data, err := yaml.Marshal(obj)
	if err != nil {
		return err
	}

	return osi.NewAtomicWriter(file).WriteFile(data)
}

func MarshalWithDelimiter(configs ...interface{}) ([]byte, error) {
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

func Matcher(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yaml" || ext == ".yml"
}
