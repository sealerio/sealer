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

package runtime

import "testing"

func TestVerionCompare(t *testing.T) {
	type args struct {
		v1 string
		v2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			"test version",
			args{
				v1: "v1.20.0",
				v2: "v1.19.1",
			},
			true,
		},
		{
			"test version",
			args{
				v1: "v1.20.0",
				v2: "v1.20.0",
			},
			true,
		},
		{
			"test version",
			args{
				v1: "v2.10.0",
				v2: "v1.20.0",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VersionCompare(tt.args.v1, tt.args.v2); got != tt.want {
				t.Errorf("VerionCompare() = %v, want %v", got, tt.want)
			}
		})
	}
}
