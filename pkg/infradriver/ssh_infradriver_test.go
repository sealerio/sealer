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

package infradriver

import (
	"net"
	"testing"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/stretchr/testify/assert"
)

func getDefaultCluster() (InfraDriver, error) {
	cluster := &v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key1=value1", "key2=value2, value3"},
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

	return NewInfraDriver(cluster)
}

func TestSSHInfraDriver_GetClusterInfo(t *testing.T) {
	driver, err := getDefaultCluster()
	if err != nil {
		assert.Error(t, err)
	}

	assert.Equal(t, driver.GetClusterName(), "my-cluster")
	assert.Equal(t, driver.GetClusterImageName(), "kubernetes:v1.19.8")
	assert.Equal(t, driver.GetClusterBasePath(), "/var/lib/sealer/data/my-cluster")
	assert.Equal(t, driver.GetClusterRootfsPath(), "/var/lib/sealer/data/my-cluster/rootfs")

	assert.Equal(t, driver.GetHostIPListByRole(common.MASTER), []net.IP{
		net.IPv4(192, 168, 0, 2),
	})

	assert.Equal(t, driver.GetHostIPListByRole(common.NODE), []net.IP{
		net.IPv4(192, 168, 0, 3),
	})

	assert.Equal(t, driver.GetHostIPList(), []net.IP{
		net.IPv4(192, 168, 0, 2),
		net.IPv4(192, 168, 0, 3),
	})

	assert.Equal(t, driver.GetClusterEnv(), map[string]string{
		"key1": "value1",
		"key2": "value2, value3",
	})

	assert.Equal(t, map[string]string{
		"HostIP":   "192.168.0.2",
		"key1":     "value1",
		"key2":     "value2, value3",
		"etcd-dir": "/data/etcd",
	}, driver.GetHostEnv(net.IPv4(192, 168, 0, 2)))

	assert.Equal(t, driver.GetHostEnv(net.IPv4(192, 168, 0, 3)), map[string]string{
		"HostIP":            "192.168.0.3",
		"key1":              "value1",
		"key2":              "value2, value3",
		"test_node_env_key": "test_node_env_value",
	})
}

func TestCheckAllHostsSameFamily(t *testing.T) {
	type args struct {
		data    []net.IP
		wanterr bool
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			"test all ipv4",
			args{
				data:    []net.IP{net.IPv4(192, 168, 0, 1), net.IPv4(192, 168, 0, 2), net.IPv4(192, 168, 0, 3)},
				wanterr: false,
			},
		},
		{
			"test all ipv6",
			args{
				data:    []net.IP{net.ParseIP("2408:4003:10bb:6a01:83b9:6360:c66d:ed57"), net.ParseIP("2408:4003:10bb:6a01:83b9:6360:c66d:ed58")},
				wanterr: false,
			},
		},
		{
			"test mixed ip address",
			args{
				data:    []net.IP{net.IPv4(192, 168, 0, 1), net.ParseIP("2408:4003:10bb:6a01:83b9:6360:c66d:ed57")},
				wanterr: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkAllHostsSameFamily(tt.args.data)
			if tt.args.wanterr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
