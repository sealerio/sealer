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

	"github.com/sealerio/sealer/logger"
)

func TestTaint_formatData(t *testing.T) {
	type fields struct {
		IPList    []string
		TaintList TaintList
	}
	type args struct {
		data string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			"1",
			fields{},
			args{
				data: "192.168.56.3 addKey1=addValue1:NoSchedule\n192.168.56.2 delKey1=delValue1:NoSchedule-\n192.168.56.3 addKey2=:NoSchedule\n192.168.56.1 delKey2=:NoSchedule-\n192.168.56.2 addKey3:NoSchedule\n192.168.56.4 delKey3:NoSchedule-\n",
			},
			false,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "192.168.56.3 addKey1==addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "192.168.56.3 addKey1=add:Value1:NoSchedule\n",
			},
			true,
		},
		{
			"no key",
			fields{},
			args{
				data: "192.168.56.3 =addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"no effect",
			fields{},
			args{
				data: "192.168.56.3 addKey1=addValue1:\n",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Taint{
				IPList:    tt.fields.IPList,
				TaintList: map[string]*taintList{},
			}
			if err := l.formatData(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("formatData(%s) error = %v, wantErr %v", tt.args.data, err, tt.wantErr)
			} else {
				logger.Info("IPList:", l.IPList)
				for k, v := range l.TaintList {
					logger.Info("[%s] taints: %v", k, *v)
				}
			}
		})
	}
}
