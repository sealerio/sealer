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

	v2 "github.com/alibaba/sealer/types/api/v2"
	//v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestEtcdBackupPlugin_Run(t *testing.T) {
	type etcdBackup struct {
		name    string
		backDir string
	}
	type args struct {
		context Context
		phase   Phase
	}

	cluster := &v2.Cluster{}
	cluster.Spec.SSH.User = "root"
	cluster.Spec.SSH.Passwd = "123456"
	for _, host := range cluster.Spec.Hosts {
		host.IPS = []string{"172.17.189.55"}
	}
	//cluster.Spec.Masters.IPList = []string{"172.17.189.55"}
	tests := []struct {
		name    string
		fields  etcdBackup
		args    args
		wantErr bool
	}{
		{
			"test label to cluster node",
			etcdBackup{
				name:    "202108112058.bak",
				backDir: "/tmp",
			},
			args{
				context: Context{
					Cluster: cluster,
				},
				phase: PhasePostInstall,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := EtcdBackupPlugin{
				name:    tt.fields.name,
				backDir: tt.fields.backDir,
			}
			if err := e.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
