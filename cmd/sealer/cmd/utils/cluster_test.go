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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/pkg/clusterfile"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func Test_ConstructClusterForScaleDown(t *testing.T) {
	data := `apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  hosts:
  - ips:
    - 192.168.0.1
    - 192.168.0.2
    - 192.168.0.3
    roles:
    - master
    ssh: {}
  - ips:
    - 192.168.0.4
    - 192.168.0.5
    - 192.168.0.6
    roles:
    - node
    ssh: {}
  image: localhost/ack-d:v1
  ssh:
    passwd: Seadent123
    pk: /root/.ssh/id_rsa
    port: "22"
    user: root
status: {}`
	type want struct {
		expectMaster []net.IP
		expectNode   []net.IP
	}

	tests := []struct {
		name            string
		want            want
		mastersToDelete []net.IP
		workersToDelete []net.IP
	}{
		{
			name: "Scale Down Cluster",
			want: want{
				expectMaster: []net.IP{net.ParseIP("192.168.0.1")},
				expectNode:   []net.IP{net.ParseIP("192.168.0.4")},
			},
			mastersToDelete: []net.IP{net.ParseIP("192.168.0.2"), net.ParseIP("192.168.0.3")},
			workersToDelete: []net.IP{net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.6")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := clusterfile.NewClusterFile([]byte(data))
			if err != nil {
				t.Errorf("failed to NewClusterFile by name,error:%v", err)
			}
			cluster := cf.GetCluster()
			if err := ConstructClusterForScaleDown(&cluster, tt.mastersToDelete, tt.workersToDelete); err != nil {
				t.Errorf("Description Failed to reduce a cluster node,error:%v", err)
			}
			var ips []net.IP
			for _, host := range cluster.Spec.Hosts {
				ips = append(ips, host.IPS...)
			}
			expectIP := append(tt.want.expectMaster, tt.want.expectNode...)
			assert.NotNil(t, ips)
			assert.Equal(t, expectIP, ips)
		})
	}
}

func Test_ConstructClusterForScaleUp(t *testing.T) {
	rawCluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key1=value1", "key2=value2;value3", "key=value"},
			SSH: v1.SSH{
				User:   "root",
				Passwd: "test123",
				Port:   "22",
			},
			Hosts: []v2.Host{
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.2")},
					Roles: []string{"master"},
					Env:   []string{"etcd-dir=/data/etcd"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.3")},
					Roles: []string{"node", "db"},
				},
			},
		},
	}
	rawCluster.APIVersion = "sealer.cloud/v2"
	rawCluster.Kind = "Cluster"
	rawCluster.Name = "mycluster"

	expectedCluster := v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key1=value1", "key2=value2;value3", "key=value"},
			SSH: v1.SSH{
				User:   "root",
				Passwd: "test123",
				Port:   "22",
			},
			Hosts: []v2.Host{
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.2")},
					Roles: []string{"master"},
					Env:   []string{"etcd-dir=/data/etcd"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.3")},
					Roles: []string{"node", "db"},
				},
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.4"), net.ParseIP("192.168.0.6")},
					Roles: []string{"master"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.7")},
					Roles: []string{"node"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
			},
		},
	}
	expectedCluster.APIVersion = "sealer.cloud/v2"
	expectedCluster.Kind = "Cluster"
	expectedCluster.Name = "mycluster"

	tests := []struct {
		name            string
		scaleFlags      *types.Flags
		joinMasters     []net.IP
		joinWorkers     []net.IP
		rawCluster      v2.Cluster
		expectedCluster v2.Cluster
	}{
		{
			name:        "test scale up to build clusters ",
			joinMasters: []net.IP{net.ParseIP("192.168.0.4"), net.ParseIP("192.168.0.6")},
			joinWorkers: []net.IP{net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.7")},
			scaleFlags: &types.Flags{
				User:     "root",
				Password: "test456",
				Port:     22,
			},
			rawCluster:      rawCluster,
			expectedCluster: expectedCluster,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ConstructClusterForScaleUp(&tt.rawCluster, tt.scaleFlags, tt.joinMasters, tt.joinWorkers); err != nil {
				t.Errorf("Scale up failed to reduce a cluster node,error:%v", err)
			}

			assert.Equal(t, tt.rawCluster.Spec.Hosts, tt.expectedCluster.Spec.Hosts)
		})
	}
}
