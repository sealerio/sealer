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
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"

	v1 "github.com/sealerio/sealer/types/api/v1"
)

func Test_getMergeConfig(t *testing.T) {
	testFileData := `apiVersion: v1
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

	wantedFileData := `apiVersion: v1
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

	filename := "/tmp/test-configmap"
	err := ioutil.WriteFile(filename, []byte(testFileData), os.ModePerm)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(filename)

	type args struct {
		path string
		data []byte
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "add namespace to each configmap",
			args: args{
				data: []byte(configmapData),
				path: filename,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMergeConfigData(tt.args.path, tt.args.data)
			if err != nil {
				assert.Errorf(t, err, "failed to MergeConfigData")
				return
			}
			assert.Equal(t, wantedFileData, string(got))
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

	filename := "/tmp/test-secret"
	err := ioutil.WriteFile(filename, []byte(testFileData), os.ModePerm)
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(filename)

	testConfig := v1.Config{
		Spec: v1.ConfigSpec{
			Data: configData,
		},
	}

	type args struct {
		config     v1.Config
		configPath string
		wanted     string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"test secret convert to file (file exist)",
			args{
				config: testConfig, configPath: filename, wanted: secretFileExistWanted},
		},
		{
			"test secret convert to file (file not exist)",
			args{testConfig, "test/secret1.yaml", secretFileNotExistWanted},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSecretYaml(tt.args.config, tt.args.configPath)
			if err != nil {
				t.Errorf("convertSecretYaml() error = %v", err)
				return
			}
			assert.Equal(t, tt.args.wanted, string(got))
		})
	}
}
