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

	v1 "github.com/sealerio/sealer/types/api/v1"
)

/*
apiVersion: sealer.aliyun.com/v1alpha1
kind: Plugin
metadata:
  name: HOSTNAME
spec:
  type: LABEL
  data: |
     192.168.0.2 master-0
     192.168.0.3 master-1
     192.168.0.4 master-2
     192.168.0.5 node-0
     192.168.0.6 node-1
     192.168.0.7 node-2
*/

func TestHostnamePlugin_Run(t *testing.T) {
	type fields struct {
		data map[string]string
	}

	type args struct {
		context Context
		phase   Phase
	}

	plugin := &v1.Plugin{}
	plugin.Spec.Data = "192.168.0.2 master-0\n192.168.0.3 master-1\n192.168.0.4 master-2\n192.168.0.5 node-0\n192.168.0.6 node-1\n192.168.0.7 node-2\n"

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test hostnameChange to cluster node",
			fields{},
			args{
				context: Context{
					Plugin: plugin,
				},
				phase: PhasePreInit,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HostnamePlugin{
				data: tt.fields.data,
			}
			if err := h.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("HostnamePlugins.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
