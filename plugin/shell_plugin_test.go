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

	typev1 "github.com/alibaba/sealer/types/api/v1"
)

/*

apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: SHELL
spec:
  action: PostInstall
  on: role=master
  data: |
     kubectl taint nodes node-role.kubernetes.io/master=:NoSchedule

*/
func TestSheller_Run(t *testing.T) {
	type args struct {
		context Context
		phase   Phase
	}

	cluster := &typev1.Cluster{}
	cluster.Spec.SSH.User = "root"
	cluster.Spec.SSH.Passwd = "7758521"
	cluster.Spec.Nodes.IPList = []string{"192.168.59.11"}

	plugin := &typev1.Plugin{}
	plugin.Spec.Data = "ifconfig"

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test shell plugin",
			args: args{
				context: Context{
					Cluster: cluster,
					Plugin:  plugin,
				},
				phase: Phase(plugin.Spec.On),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Sheller{}
			if err := s.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
