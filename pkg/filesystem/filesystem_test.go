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

package filesystem

import (
	"testing"

	"github.com/alibaba/sealer/pkg/runtime"

	v2 "github.com/alibaba/sealer/types/api/v2"

	k8sV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/alibaba/sealer/types/api/v1"
)

var testCluster = &v2.Cluster{
	ObjectMeta: k8sV1.ObjectMeta{Name: "my-cluster"},
	Spec: v2.ClusterSpec{
		Image: "kubernetes:v1.19.9",
		Hosts: []v2.Host{
			{
				IPS: []string{
					"192.168.56.111",
				},
				Roles: []string{"master"},
			}, {
				IPS: []string{
					"192.168.56.112",
				},
				Roles: []string{"node"},
			},
		},
		SSH: v1.SSH{
			User:   "root",
			Passwd: "******",
		},
	},
}

func TestMount(t *testing.T) {
	type args struct {
		cluster *v2.Cluster
	}
	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "test mount",
			arg: args{
				cluster: testCluster,
			},
			wantErr: false,
		},
	}
	rootfs := "/var/lib/sealer/data/overlay2/"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem, err := NewFilesystem(rootfs)
			if err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}

			if err = fileSystem.MountRootfs(tt.arg.cluster, append(runtime.GetMasterIPList(testCluster), runtime.GetNodeIPList(testCluster)...), true); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}

func TestUnMount(t *testing.T) {
	type args struct {
		cluster *v2.Cluster
	}
	tests := []struct {
		name    string
		arg     args
		wantErr bool
	}{
		{
			name: "test unmount",
			arg: args{
				cluster: testCluster,
			},
			wantErr: false,
		},
	}
	rootfs := "/var/lib/sealer/data/overlay2/"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem, err := NewFilesystem(rootfs)
			if err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
			if err = fileSystem.UnMountRootfs(tt.arg.cluster, append(tt.arg.cluster.GetMasterIPList(), tt.arg.cluster.GetNodeIPList()...)); err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
}
