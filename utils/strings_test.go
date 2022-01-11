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
	"reflect"
	"testing"
)

func TestAppendIPList(t *testing.T) {
	type args struct {
		src []string
		dst []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"test merge ip list",
			args{
				src: []string{"172.16.0.149", "172.16.0.181", "172.16.0.180"},
				dst: []string{"172.16.0.181", "172.16.0.182", "172.16.0.181", "172.16.0.183", "172.16.0.149"},
			},
			[]string{"172.16.0.149", "172.16.0.181", "172.16.0.180", "172.16.0.182", "172.16.0.183"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendDiffSlice(tt.args.src, tt.args.dst); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendDiffSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
