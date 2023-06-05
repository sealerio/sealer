// Copyright © 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package clusterruntime

import (
	"net"
	"reflect"
	"testing"

	containerruntime "github.com/sealerio/sealer/pkg/container-runtime"
	"github.com/sealerio/sealer/pkg/infradriver"
	"github.com/sealerio/sealer/pkg/registry"
	"github.com/sealerio/sealer/pkg/runtime"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"

	"github.com/stretchr/testify/assert"
)

func TestInstaller_GetCurrentDriver(t *testing.T) {
	type fields struct {
		RuntimeConfig             RuntimeConfig
		infraDriver               infradriver.InfraDriver
		containerRuntimeInstaller containerruntime.Installer
		clusterRuntimeType        string
		hooks                     map[Phase]HookConfigList
		regConfig                 v2.Registry
	}
	var tests []struct {
		name    string
		fields  fields
		want    registry.Driver
		want1   runtime.Driver
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{
				RuntimeConfig:             tt.fields.RuntimeConfig,
				infraDriver:               tt.fields.infraDriver,
				containerRuntimeInstaller: tt.fields.containerRuntimeInstaller,
				clusterRuntimeType:        tt.fields.clusterRuntimeType,
				hooks:                     tt.fields.hooks,
				regConfig:                 tt.fields.regConfig,
			}
			got, got1, err := i.GetCurrentDriver()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentDriver() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCurrentDriver() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("GetCurrentDriver() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestInstaller_setNodeLabels(t *testing.T) {
	type fields struct {
		RuntimeConfig             RuntimeConfig
		infraDriver               infradriver.InfraDriver
		containerRuntimeInstaller containerruntime.Installer
		clusterRuntimeType        string
		hooks                     map[Phase]HookConfigList
		regConfig                 v2.Registry
	}
	type args struct {
		hosts  []net.IP
		driver runtime.Driver
	}
	var tests []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Installer{
				RuntimeConfig:             tt.fields.RuntimeConfig,
				infraDriver:               tt.fields.infraDriver,
				containerRuntimeInstaller: tt.fields.containerRuntimeInstaller,
				clusterRuntimeType:        tt.fields.clusterRuntimeType,
				hooks:                     tt.fields.hooks,
				regConfig:                 tt.fields.regConfig,
			}
			if err := i.setNodeLabels(tt.args.hosts, tt.args.driver); (err != nil) != tt.wantErr {
				t.Errorf("setNodeLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_chooseCRIInstaller(t *testing.T) {
	type args struct {
		containerRuntime string
	}
	tests := []struct {
		name    string
		args    args
		want    v2.ContainerRuntimeConfig
		wantErr bool
	}{
		{
			name: "test for choose docker",
			args: args{
				containerRuntime: "docker",
			},
			want:    v2.ContainerRuntimeConfig{Type: "docker"},
			wantErr: false,
		},
		{
			name: "test for choose containerd",
			args: args{
				containerRuntime: "containerd",
			},
			want:    v2.ContainerRuntimeConfig{Type: "containerd"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			infraDriver, err := getDefaultCluster()
			if err != nil {
				assert.Error(t, err)
			}

			got, err := getCRIInstaller(tt.args.containerRuntime, infraDriver)
			if err != nil {
				t.Errorf("chooseCRIInstaller() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			info, err := got.GetInfo()
			if err != nil {
				assert.Error(t, err)
			}

			if !reflect.DeepEqual(info.Type, tt.want.Type) {
				t.Errorf("chooseCRIInstaller() got = %v, want %v", info.Type, tt.want)
			}
		})
	}
}

func getDefaultCluster() (infradriver.InfraDriver, error) {
	cluster := &v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key1=value1", "key2=value2;value3"},
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
					Env:   []string{"test_node_env_key=test_node_env_value"},
				},
			},
		},
	}
	cluster.APIVersion = "sealer.io/v2"
	cluster.Kind = "Cluster"
	cluster.Name = "my-cluster"

	return infradriver.NewInfraDriver(cluster)
}
