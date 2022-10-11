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

	"github.com/sealerio/sealer/cmd/sealer/cmd/types"
	"github.com/sealerio/sealer/pkg/clusterfile"
	"github.com/stretchr/testify/assert"
)

func Test_returnFilteredIPList(t *testing.T) {
	tests := []struct {
		name              string
		clusterIPList     []net.IP
		toBeDeletedIPList []net.IP
		IPListExpected    []net.IP
	}{
		{
			"test",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
		},
		{
			"test1",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
		},
		{
			"test2",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3")},
		},
		{
			"test3",
			[]net.IP{},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := removeIPList(tt.clusterIPList, tt.toBeDeletedIPList); res != nil {
				assert.Equal(t, tt.IPListExpected, res)
			}
		})
	}
}

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
	data := `apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  hosts:
  - ips:
    - 172.16.0.230
    roles:
    - master
    ssh: {}
  - ips:
    - 172.16.0.233
    roles:
    - node
    ssh: {}`

	type want struct {
		expectMaster []net.IP
		expectNode   []net.IP
	}

	tests := []struct {
		name string
		//scaleFlags  *types.Flags
		joinMasters []net.IP
		joinWorkers []net.IP
		want        want
	}{
		{
			name:        "test scale up to build clusters ",
			joinMasters: []net.IP{net.ParseIP("172.16.0.231"), net.ParseIP("172.16.0.232")},
			joinWorkers: []net.IP{net.ParseIP("172.16.0.234"), net.ParseIP("172.16.0.235")},
			want: want{
				expectMaster: []net.IP{net.ParseIP("172.16.0.230"), net.ParseIP("172.16.0.231"), net.ParseIP("172.16.0.232")},
				expectNode:   []net.IP{net.ParseIP("172.16.0.233"), net.ParseIP("172.16.0.234"), net.ParseIP("172.16.0.235")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := clusterfile.NewClusterFile([]byte(data))
			if err != nil {
				t.Errorf("failed to NewClusterFile by name,error:%v", err)
			}
			cluster := cf.GetCluster()
			if err := ConstructClusterForScaleUp(&cluster, &types.Flags{}, tt.joinMasters, tt.joinWorkers); err != nil {
				t.Errorf("Scale up failed to reduce a cluster node,error:%v", err)
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
