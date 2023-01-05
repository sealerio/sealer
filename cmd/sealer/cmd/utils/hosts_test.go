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

	v1 "github.com/sealerio/sealer/types/api/v1"

	"github.com/stretchr/testify/assert"

	"github.com/sealerio/sealer/pkg/clusterfile"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func Test_TransferIPStrToHosts(t *testing.T) {
	data := `apiVersion: sealer.io/v2
kind: Cluster
metadata:
  creationTimestamp: null
  name: my-cluster
spec:
  hosts:
  - ips:
    - 192.168.0.5
    - 192.168.0.6
    - 192.168.0.7
    roles:
    - master
    ssh: {}
  - ips:
    - 192.168.0.4
    - 192.168.0.3
    - 192.168.0.2
    roles:
    - node
    ssh: {}
`
	type ages struct {
		inMasters string
		inNodes   string
	}
	tests := []struct {
		name    string
		args    ages
		want    []v2.Host
		wantErr bool
	}{
		{
			name: "test getHosts",
			args: ages{
				inMasters: "192.168.0.5,192.168.0.6,192.168.0.7",
				inNodes:   "192.168.0.4,192.168.0.3,192.168.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masterIPList, nodeIPList, err := ParseToNetIPList(tt.args.inMasters, tt.args.inNodes)
			if err != nil {
				t.Error(err)
				return
			}
			hosts := TransferIPToHosts(masterIPList, nodeIPList, v1.SSH{})
			cf, err := clusterfile.NewClusterFile([]byte(data))
			if err != nil {
				assert.Errorf(t, err, "failed to NewClusterFile by name")
			}
			cluster := cf.GetCluster()
			var ips net.IP
			for _, host := range cluster.Spec.Hosts {
				for _, ips = range host.IPS {
					assert.NotNil(t, ips)
				}
			}

			var i net.IP
			for _, ip := range hosts {
				for _, i = range ip.IPS {
					assert.NotNil(t, i)
				}
			}
			assert.Equal(t, ips, i)
		})
	}
}
