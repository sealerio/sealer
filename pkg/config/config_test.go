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
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getMergeConfig(t *testing.T) {
	testSrcData := `apiVersion: v1
data:
  key1: myConfigMap1
kind: ConfigMap
metadata:
  name: myConfigMap1
---
apiVersion: v1
data:
  key2: myConfigMap2
kind: ConfigMap
metadata:
  name: myConfigMap2
---
apiVersion: v1
data:
  key3: myConfigMap3
kind: ConfigMap
metadata:
  name: myConfigMap3
`

	wantedData := `apiVersion: v1
data:
    key1: myConfigMap1
    test-key: test-key
kind: ConfigMap
metadata:
    name: myConfigMap1
    namespace: test-namespace
---
apiVersion: v1
data:
    key2: myConfigMap2
    test-key: test-key
kind: ConfigMap
metadata:
    name: myConfigMap2
    namespace: test-namespace
---
apiVersion: v1
data:
    key3: myConfigMap3
    test-key: test-key
kind: ConfigMap
metadata:
    name: myConfigMap3
    namespace: test-namespace
`

	configmapData := `data:
  test-key: test-key
metadata:
  namespace: test-namespace
`

	type args struct {
		src        []byte
		configData []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add namespace to each configmap",
			args: args{
				configData: []byte(configmapData),
				src:        []byte(testSrcData),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMergeConfigData(tt.args.src, tt.args.configData)
			if err != nil {
				assert.Errorf(t, err, "failed to MergeConfigData")
				return
			}
			assert.Equal(t, wantedData, string(got))
		})
	}
}

func Test_convertSecretYaml(t *testing.T) {
	configData := `global: e2FiYzogeHh4fQo=
components: e215c3FsOntjcHU6e3JlcXVlc3Q6IDEwMDBtfX19Cg==`

	testFileData := `apiVersion: v1
data:
kind: Secret
metadata:
  name: gu-demo-configuration
  namespace: default
type: Opaque`

	secretFileExistWanted := `apiVersion: v1
data:
  components: ZTIxNWMzRnNPbnRqY0hVNmUzSmxjWFZsYzNRNklERXdNREJ0ZlgxOUNnPT0=
  global: ZTJGaVl6b2dlSGg0ZlFvPQ==
kind: Secret
metadata:
  creationTimestamp: null
  name: gu-demo-configuration
  namespace: default
type: Opaque
`

	secretFileNotExistWanted := `data:
  components: ZTIxNWMzRnNPbnRqY0hVNmUzSmxjWFZsYzNRNklERXdNREJ0ZlgxOUNnPT0=
  global: ZTJGaVl6b2dlSGg0ZlFvPQ==
metadata:
  creationTimestamp: null
`

	type args struct {
		src        []byte
		configData []byte
		wanted     []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"test secret convert with src",
			args{
				src: []byte(testFileData), configData: []byte(configData), wanted: []byte(secretFileExistWanted)},
		},
		{
			"test secret convert without src",
			args{src: nil, configData: []byte(configData), wanted: []byte(secretFileNotExistWanted)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSecretYaml(tt.args.src, tt.args.configData)
			if err != nil {
				t.Errorf("convertSecretYaml() error = %v", err)
				return
			}
			assert.Equal(t, tt.args.wanted, got)
		})
	}
}
