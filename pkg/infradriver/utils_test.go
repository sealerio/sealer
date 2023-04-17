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

package infradriver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	k8sv1 "k8s.io/api/core/v1"
)

func TestFormatData(t *testing.T) {
	type args struct {
		data   string
		wanted k8sv1.Taint
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			"test format date: key1=value1:NoSchedule",
			args{
				data: "key1=value1:NoSchedule",
				wanted: k8sv1.Taint{
					Key:    "key1",
					Value:  "value1",
					Effect: k8sv1.TaintEffect("NoSchedule"),
				},
			},
		},
		{
			"test format date: key2:PreferNoSchedule",
			args{
				data: "key2=:PreferNoSchedule",
				wanted: k8sv1.Taint{
					Key:    "key2",
					Value:  "",
					Effect: k8sv1.TaintEffect("PreferNoSchedule"),
				},
			},
		},
		{
			"test format date: key3=:NoExecute",
			args{
				data: "key3=:NoExecute",
				wanted: k8sv1.Taint{
					Key:    "key3",
					Value:  "",
					Effect: k8sv1.TaintEffect("NoExecute"),
				},
			},
		},
		{
			"test format date: key4:NoExecute-",
			args{
				data: "key4:NoExecute-",
				wanted: k8sv1.Taint{
					Key:    "key4",
					Value:  "",
					Effect: k8sv1.TaintEffect("NoExecute-"),
				},
			},
		},
		{
			"test format date: key7-",
			args{
				data: "key7-",
				wanted: k8sv1.Taint{
					Key:    "key7-",
					Value:  "",
					Effect: k8sv1.TaintEffect(""),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := formatData(tt.args.data)
			if err != nil {
				t.Errorf("failed to format data, error:%v", err)
			}
			assert.Equal(t, tt.args.wanted, result)
		})
	}
}
