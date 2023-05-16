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
	imagev1 "github.com/sealerio/sealer/pkg/define/image/v1"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_ConstructClusterForScaleDown(t *testing.T) {
	data := `apiVersion: sealer.io/v2
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
	rawCluster.APIVersion = "sealer.io/v2"
	rawCluster.Kind = "Cluster"
	rawCluster.Name = "mycluster"

	expectedCluster := v2.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "sealer.io/v2",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "mycluster",
		},
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

	type testT struct {
		name            string
		scaleFlags      *types.ScaleUpFlags
		currentNodes    []net.IP
		joinMasters     []net.IP
		joinWorkers     []net.IP
		rawCluster      v2.Cluster
		expectedCluster v2.Cluster
	}
	t1 := testT{
		name:        "test scale up to build clusters ",
		joinMasters: []net.IP{net.ParseIP("192.168.0.4"), net.ParseIP("192.168.0.6")},
		joinWorkers: []net.IP{net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.7")},
		scaleFlags: &types.ScaleUpFlags{
			User:     "root",
			Password: "test456",
			Port:     22,
		},
		rawCluster:      rawCluster,
		expectedCluster: expectedCluster,
	}
	t.Run(t1.name, func(t *testing.T) {
		mj, nj, err := ConstructClusterForScaleUp(&t1.rawCluster, t1.scaleFlags, t1.currentNodes, t1.joinMasters, t1.joinWorkers)
		if err != nil {
			t.Errorf("Scale up failed to reduce a cluster node,error:%v", err)
		}

		assert.Equal(t, t1.joinMasters, mj)
		assert.Equal(t, t1.joinWorkers, nj)
		assert.Equal(t, t1.rawCluster.Spec.Hosts, t1.expectedCluster.Spec.Hosts)
	})

	expectedCluster = v2.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "sealer.io/v2",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "mycluster",
		},
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
					IPS:   []net.IP{net.ParseIP("192.168.0.6")},
					Roles: []string{"master"},
					SSH: v1.SSH{
						User:   "root",
						Passwd: "test456",
						Port:   "22",
					},
				},
				{
					IPS:   []net.IP{net.ParseIP("192.168.0.5")},
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

	t1 = testT{
		name:         "test scale up to build clusters ",
		currentNodes: []net.IP{net.ParseIP("192.168.0.4"), net.ParseIP("192.168.0.7")},
		joinMasters:  []net.IP{net.ParseIP("192.168.0.4"), net.ParseIP("192.168.0.6")},
		joinWorkers:  []net.IP{net.ParseIP("192.168.0.5"), net.ParseIP("192.168.0.7")},
		scaleFlags: &types.ScaleUpFlags{
			User:     "root",
			Password: "test456",
			Port:     22,
		},
		rawCluster:      rawCluster,
		expectedCluster: expectedCluster,
	}
	t.Run(t1.name, func(t *testing.T) {
		mj, nj, err := ConstructClusterForScaleUp(&t1.rawCluster, t1.scaleFlags, t1.currentNodes, t1.joinMasters, t1.joinWorkers)
		if err != nil {
			t.Errorf("Scale up failed to reduce a cluster node,error:%v", err)
		}

		assert.Equal(t, []net.IP{net.ParseIP("192.168.0.6")}, mj)
		assert.Equal(t, []net.IP{net.ParseIP("192.168.0.5")}, nj)
		assert.Equal(t, t1.rawCluster.Spec.Hosts, t1.expectedCluster.Spec.Hosts)
	})
}

func Test_MergeClusterWithImageExtension(t *testing.T) {
	rawCluster := &v2.Cluster{
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"key=value", "key1=value1", "key2=value2"},
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
	rawCluster.APIVersion = "sealer.io/v2"
	rawCluster.Kind = "Cluster"
	rawCluster.Name = "mycluster"

	extension := imagev1.ImageExtension{
		Env: map[string]string{"KeyDefault": "ValueDefault"},
	}

	expectedCluster := &v2.Cluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "sealer.io/v2",
			Kind:       "Cluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "mycluster",
		},
		Spec: v2.ClusterSpec{
			Image: "kubernetes:v1.19.8",
			Env:   []string{"KeyDefault=ValueDefault", "key=value", "key1=value1", "key2=value2"},
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

	tests := []struct {
		name       string
		rawCluster *v2.Cluster
		imageExt   imagev1.ImageExtension
	}{
		{
			name:       " test merge image extension with v2.cluster",
			rawCluster: rawCluster,
			imageExt:   extension,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mergedWithExt := MergeClusterWithImageExtension(tt.rawCluster, tt.imageExt)

			assert.Equal(t, mergedWithExt, expectedCluster)
		})
	}
}
