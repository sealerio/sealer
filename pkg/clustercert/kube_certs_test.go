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

package clustercert

import (
	"net"
	"testing"
)

func TestGenerateAll(t *testing.T) {
	basePath := "/tmp/kubernetes/pki"
	etcdBasePath := "/tmp/kubernetes/pki/etcd"
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			"generate all certs",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := GenerateAllKubernetesCerts(basePath, etcdBasePath, "master1", "10.64.0.0/10", "cluster.local", []string{"test.com", "192.168.1.2", "kubernetes.default.svc.sealyun"}, net.ParseIP("172.27.139.11")); (err != nil) != tt.wantErr {
				t.Errorf("GenerateAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateAPIServerCert(t *testing.T) {
	tests := []struct {
		pkiPath  string
		certSans []string
		name     string
		wantErr  bool
	}{
		{"/tmp/kubernetes/pki",
			[]string{"kaka.com", "apiserver.cluster.local", "192.168.1.100"},
			"Update APIServer Cert sans",
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := UpdateAPIServerCertSans(tt.pkiPath, tt.certSans); (err != nil) != tt.wantErr {
				t.Errorf("UpdateAPIServerCert() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
