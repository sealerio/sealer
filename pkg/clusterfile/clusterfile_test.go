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
	"testing"

	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/stretchr/testify/assert"

	"github.com/sealerio/sealer/common"
)

func TestSaveAll(t *testing.T) {
	cluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key1=value1", "key2=value2;value3", "key=value"},
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
	cluster.APIVersion = "sealer.cloud/v2"
	cluster.Kind = "Cluster"
	cluster.Name = "my-cluster"

	plugin1 := v1.Plugin{
		Spec: v1.PluginSpec{
			Type:   "HOSTNAME",
			Data:   "192.168.0.2 master-0\n",
			Action: "PreInit",
		},
	}
	plugin1.Name = "MyHostname"
	plugin1.Kind = common.Plugin
	plugin1.APIVersion = APIVersion

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
	plugin2.APIVersion = "sealer.aliyun.com/v1alpha1"

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
		data   []byte
		wanted wanted
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			"test decode cluster file",
			args{
				data: []byte(data),
				wanted: wanted{
					cluster: cluster,
					config:  []v1.Config{config},
					plugins: []v1.Plugin{plugin1, plugin2}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := NewClusterFile(tt.args.data)
			if err != nil {
				t.Errorf("failed to decode cluster file, error is:(%v)", err)
			}

			cluster := cf.GetCluster()
			env := "key=value"
			cluster.Spec.Env = append(cluster.Spec.Env, env)
			cf.SetCluster(cluster)
			assert.NotNil(t, cf)

			assert.Equal(t, tt.args.wanted.cluster, cf.GetCluster())

			assert.Equal(t, tt.args.wanted.config, cf.GetConfigs())

			assert.Equal(t, tt.args.wanted.plugins, cf.GetPlugins())

			kubeadm := cf.GetKubeadmConfig()
			assert.NotNil(t, kubeadm)

			assert.Equal(t, kubeadm.InitConfiguration.TypeMeta.Kind, common.InitConfiguration)
			assert.Equal(t, kubeadm.KubeProxyConfiguration.TypeMeta.Kind, common.KubeProxyConfiguration)
		})
	}
}
