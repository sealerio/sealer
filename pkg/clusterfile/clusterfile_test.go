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

package clusterfile

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"

	"github.com/sealerio/sealer/common"
)

var defaultHA = true
var defaultInsecure = false

func TestSaveAll(t *testing.T) {
	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			Image:    "kubernetes:v1.19.8",
			DataRoot: "/var/lib/sealer/data",
			Env: []string{"key1=value1", "key2=value2;value3", "key=value",
				"LocalRegistryDomain=sea.hub", "LocalRegistryPort=5000", "LocalRegistryURL=sea.hub:5000",
				"RegistryDomain=sea.hub", "RegistryPort=5000", "RegistryURL=sea.hub:5000"},
			Registry: v2.Registry{
				LocalRegistry: &v2.LocalRegistry{
					RegistryConfig: v2.RegistryConfig{
						Domain: "sea.hub",
						Port:   5000,
					},
					HA:       &defaultHA,
					Insecure: &defaultInsecure,
				},
			},
			SSH: v1.SSH{
				User:     "root",
				Passwd:   "test123",
				Port:     "22",
				Pk:       "xxx",
				PkPasswd: "xxx",
			},
			Hosts: []v2.Host{
				{
					IPS:   []net.IP{net.IPv4(192, 168, 0, 2)},
					Roles: []string{"master"},
					Env:   []string{"etcd-dir=/data/etcd"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.IPv4(192, 168, 0, 3)},
					Roles: []string{"node", "db"},
				},
			},
		},
	}
	cluster.APIVersion = "sealer.io/v2"
	cluster.Kind = "Cluster"
	cluster.Name = "my-cluster"

	plugin2 := v1.Plugin{
		Spec: v1.PluginSpec{
			Type:   "SHELL",
			Data:   "kubectl get nodes\n",
			Scope:  "master",
			Action: "PostInstall",
		},
	}
	plugin2.Name = "MyShell"
	plugin2.Kind = "Plugin"
	plugin2.APIVersion = "sealer.io/v1"

	config := v1.Config{
		Spec: v1.ConfigSpec{
			Path: "etc/mysql.yaml",
			Data: "mysql-user: root\nmysql-passwd: xxx\n",
		},
	}
	config.Name = "mysql-config"
	config.Kind = "Config"
	config.APIVersion = "sealer.com/v1alpha1"

	type wanted struct {
		cluster v2.Cluster
		config  []v1.Config
		plugins []v1.Plugin
	}

	type args struct {
		wanted wanted
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			name: "test decode cluster file",
			args: args{
				wanted: wanted{
					cluster: cluster,
					config:  []v1.Config{config},
					plugins: []v1.Plugin{plugin2},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clusterFile := &ClusterFile{cluster: &cluster, configs: []v1.Config{config}, plugins: []v1.Plugin{plugin2}}
			clusterFilePath := common.GetDefaultClusterfile()
			if err := os.MkdirAll(filepath.Dir(clusterFilePath), common.FileMode0755); err != nil {
				t.Errorf("failed to create directory, error is:(%v)", err)
			}
			if err := clusterFile.SaveAll(SaveOptions{}); err != nil {
				t.Errorf("failed to save all file, error is:(%v)", err)
			}
			clusterFileData, err := os.ReadFile(filepath.Clean(clusterFilePath))
			if err != nil {
				t.Errorf("failed to read cluster file, error is:(%v)", err)
			}

			cf, err := NewClusterFile(clusterFileData)
			if err != nil {
				t.Errorf("failed to get clusterfile interface, error is:(%v)", err)
			}

			assert.Equal(t, tt.args.wanted.config, cf.GetConfigs())
			assert.Equal(t, tt.args.wanted.plugins, cf.GetPlugins())
			assert.Equal(t, tt.args.wanted.cluster, cf.GetCluster())

			if err := os.Remove(clusterFilePath); err != nil {
				t.Errorf("failed to remove clusterfile, error is:(%v)", err)
			}
		})
	}
}
