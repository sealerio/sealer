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

	v1 "github.com/alibaba/sealer/types/api/v1"
)

func TestLabelsNodes_Run(t *testing.T) {
	type fields struct {
		data map[string][]label
	}

	plugin := &v1.Plugin{}
	plugin.Spec.Data = "192.9.200.21 ws=demo21\n192.9.200.22 ssd=true\n192.9.200.23 ssd=true,hdd=false"

	type args struct {
		context Context
		phase   Phase
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"test label to cluster node",
			fields{},
			args{
				context: Context{
					Plugin: plugin,
				},
				phase: PhasePostInstall,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := LabelsNodes{
				data: tt.fields.data,
			}
			if err := l.Run(tt.args.context, tt.args.phase); (err != nil) != tt.wantErr {
				t.Errorf("LabelsNodes.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
