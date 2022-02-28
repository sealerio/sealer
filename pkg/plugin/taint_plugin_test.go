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

	"github.com/alibaba/sealer/logger"
	v1 "k8s.io/api/core/v1"
)

func TestTaint_formatData(t *testing.T) {
	type fields struct {
		DelTaintList []v1.Taint
		AddTaintList []v1.Taint
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
				data: "addKey1=addValue1:NoSchedule\ndelKey1=delValue1:NoSchedule-\naddKey2=:NoSchedule\ndelKey2=:NoSchedule-;addKey3:NoSchedule;delKey3:NoSchedule-\n",
			},
			false,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "addKey1==addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"invalid taint argument",
			fields{},
			args{
				data: "addKey1=add:Value1:NoSchedule\n",
			},
			true,
		},
		{
			"no key",
			fields{},
			args{
				data: "=addValue1:NoSchedule\n",
			},
			true,
		},
		{
			"no effect",
			fields{},
			args{
				data: "addKey1=addValue1:\n",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := Taint{
				DelTaintList: tt.fields.DelTaintList,
				AddTaintList: tt.fields.AddTaintList,
			}
			if err := l.formatData(tt.args.data); (err != nil) != tt.wantErr {
				t.Errorf("formatData() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				logger.Info(l.DelTaintList)
				logger.Info(l.AddTaintList)
			}
		})
	}
}
