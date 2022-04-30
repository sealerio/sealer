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

package plugin

import (
	"testing"

	"github.com/sealerio/sealer/common"
	v1 "github.com/sealerio/sealer/types/api/v1"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func TestEtcdBackupPlugin_Run(t *testing.T) {
	type args struct {
		context Context
		phase   Phase
	}

	cluster := &v2.Cluster{}
	cluster.Spec.SSH.User = "root"
	cluster.Spec.SSH.Passwd = "123456"
	cluster.Spec.Hosts = []v2.Host{
		{
			IPS:   []string{"192.168.0.2"},
			Roles: []string{common.MASTER},
		},
	}
	plugin := v1.Plugin{}
	plugin.Spec.On = "/tmp/202108112058.bak"
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test label to cluster node",
			args{
				context: Context{
					Cluster: cluster,
					Plugin:  &plugin,
				},
				phase: PhasePostInstall,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EtcdBackupPlugin{}
			if err := e.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
