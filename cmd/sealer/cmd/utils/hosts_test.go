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
	"fmt"
	"testing"

	v2 "github.com/sealerio/sealer/types/api/v2"
)

func Test_getHosts(t *testing.T) {
	type ages struct {
		inMasters string
		inNodes   string
	}
	tests := []struct {
		name    string
		args    ages
		want    []v2.Host
		wantErr bool
	}{
		{
			name: "test getHosts",
			args: ages{
				inMasters: "192.168.0.5,192.168.0.6,192.168.0.7",
				inNodes:   "192.168.0.4,192.168.0.3,192.168.0.2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hosts, err := GetHosts(tt.args.inMasters, tt.args.inNodes)
			if err != nil {
				t.Error(err)
				return
			}
			fmt.Println(hosts)
		})
	}
}
