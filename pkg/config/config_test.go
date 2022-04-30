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
	"testing"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/sealerio/sealer/utils"
)

func TestDumper_Dump(t *testing.T) {
	type fields struct {
		configs     []v1.Config
		clusterName string
	}
	type args struct {
		clusterfile string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"test dump clusterfile configs",
			fields{
				configs:     nil,
				clusterName: "my-cluster",
			},
			args{clusterfile: "test/test_clusterfile.yaml"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Dumper{
				Configs: tt.fields.configs,
				Cluster: &v2.Cluster{},
			}
			c.Cluster.Name = tt.fields.clusterName
			configs, err := utils.DecodeV1CRD(tt.args.clusterfile, common.Config)
			if err != nil {
				t.Error(err)
				return
			}
			if err := c.Dump(configs.([]v1.Config)); (err != nil) != tt.wantErr {
				t.Errorf("Dump() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getMergeConfig(t *testing.T) {
	type args struct {
		path string
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test",
			args: args{
				data: []byte("spec:\n  image: kubernetes:v1.19.8"),
				path: "test/test_clusterfile.yaml",
			},
		}, {
			name: "test",
			args: args{
				data: []byte("spec:\n  template:\n    metadata:\n      labels:\n        name: tigera-operatorssssss"),
				path: "test/tigera-operator.yaml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getMergeConfigData(tt.args.path, tt.args.data)
			if err != nil {
				t.Error(err)
				return
			}
			err = ioutil.WriteFile("test_"+tt.args.path, got, common.FileMode0644)
			if err != nil {
				t.Error(err)
			}
		})
	}
}

func Test_convertSecretYaml(t *testing.T) {
	testConfig := v1.Config{}
	testConfig.Spec.Data = `
global: e2FiYzogeHh4fQo=
components: e215c3FsOntjcHU6e3JlcXVlc3Q6IDEwMDBtfX19Cg==`
	testConfig.Spec.Process = "value|toJson|toBase64|toSecret"
	type args struct {
		config     v1.Config
		configPath string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"test secret convert to file (file exist)",
			args{testConfig, "test/secret.yaml"},
		},
		{
			"test secret convert to file (file not exist)",
			args{testConfig, "test/secret1.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertSecretYaml(tt.args.config, tt.args.configPath)
			if err != nil {
				t.Errorf("convertSecretYaml() error = %v", err)
				return
			}
			fmt.Println(string(got))
		})
	}
}
