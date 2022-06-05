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

package apply

import (
	"testing"

	"github.com/sealerio/sealer/utils/net"
	"github.com/sirupsen/logrus"
)

func TestAssemblyIPList(t *testing.T) {
	tests := []struct {
		name    string
		args    *Args
		wantErr bool
	}{
		{
			"baseData",
			&Args{
				Masters:    "10.110.101.1-10.110.101.5",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			false,
		},
		{
			"errorData",
			&Args{
				Masters:    "10.110.101.10-10.110.101.5",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
		{
			"errorData2",
			&Args{
				Masters:    "10.110.101.10-10.110.101.5-10.110.101.55",
				Nodes:      "10.110.101.1-10.110.101.5",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
		{
			"errorData3",
			&Args{
				Masters:    "-10.110.101.",
				Nodes:      "10.110.101.1-",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
		{
			"errorData4",
			&Args{
				Masters:    "a-b",
				Nodes:      "a-",
				User:       "",
				Password:   "",
				Pk:         "",
				PkPassword: "",
				PodCidr:    "",
				SvcCidr:    "",
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := net.AssemblyIPList(&tt.args.Masters); (err != nil) != tt.wantErr {
				logrus.Errorf("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
			}
			logrus.Infof("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
		})
	}
}
