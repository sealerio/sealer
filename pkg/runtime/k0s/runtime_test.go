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

package k0s

import (
	"testing"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/sealerio/sealer/pkg/runtime/k0s/k0sctl/v1beta1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

const (
	mockClusterFile = `
apiVersion: sealer.cloud/v2
kind: Cluster
metadata:
  name: my-k0s-cluster
spec:
  image: k0s:v1.24.1-rc
  ssh:
    passwd: test123
    port: "2222"
  hosts:
    - ips: [ 192.168.0.2 ] # this master ssh port is different with others.
      roles: [ master ]
      ssh:
        passwd: yyy
        port: "22"
    - ips: [ 192.168.0.3,192.168.0.4 ]
      roles: [ worker ]
    - ips: [ 192.168.0.5 ]
      roles: [ worker ]
`
	mockK0sctlYaml = `
apiVersion: k0sctl.k0sproject.io/v1beta1
kind: Cluster
metadata:
  name: my-k0s-cluster
spec:
  hosts:
  - role: controller
    installFlags:
    - --debug
    ssh:
      address: 10.0.0.1
      user: root
      port: 22
      keyPath: ~/.ssh/id_rsa
  - role: worker
    installFlags:
    - --debug
    ssh:
      address: 10.0.0.2
  k0s:
    version: 0.10.0
    config:
      apiVersion: k0s.k0sproject.io/v1beta1
      kind: Cluster
      metadata:
        name: my-k0s-cluster
      spec:
        images:
          calico:
            cni:
              image: calico/cni
              version: v3.16.2
`
)

func TestNewK0sRuntime(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "test newK0sRuntime",
			wantErr: false,
		},
		{
			name:    "test newK0sRuntime",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClusterFile := v2.Cluster{}
			testClusterConfigYAML := v1beta1.Cluster{}
			if err := yaml.Unmarshal([]byte(mockClusterFile), &testClusterFile); err != nil {
				logrus.Errorf("decode to clusterfile yaml failed: %s", err)
			}
			if err := yaml.Unmarshal([]byte(mockK0sctlYaml), &testClusterConfigYAML); err != nil {
				logrus.Errorf("decode to k0sctl.yaml failed: %s", err)
			}
			_, err := NewK0sRuntime(&testClusterFile)
			if err != nil {
				t.Errorf("NewK0sRuntime err: %s", err)
			}
		})
	}
}
