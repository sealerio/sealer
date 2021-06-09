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

package guest

import (
	"fmt"
	"testing"
)

func TestCalico_Manifests(t *testing.T) {
	type args struct {
		metadata MetaData
		template string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test default calico template",
			args: args{
				metadata: MetaData{
					Interface: "interface=en.*|eth.*",
					CIDR:      "192.168.0.0/16",
					IPIP:      true,
					MTU:       "1440",
				},
				template: "",
			},
		},
		{
			name: "test custom calico template",
			args: args{
				metadata: MetaData{
					Interface: "can-reach=192.168.0.1",
					CIDR:      "",
					IPIP:      true,
					MTU:       "1440",
				},
				template: `	MTU:{{ .MTU }}
							Interface:{{.Interface}}
							IPIP:{{if not .IPIP }}Off{{else}}Always{{end}}
							VXLAN:{{if .IPIP }}Off{{else if .VXLAN}}Always{{else}}Never{{end}}
							CIDR:{{ .CIDR }}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calico := NewNetWork(CALICO, tt.args.metadata)
			fmt.Println(calico.Manifests(tt.args.template))
		})
	}
}
