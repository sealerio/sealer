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
	"github.com/alibaba/sealer/common"
	"github.com/alibaba/sealer/pkg/logger"
	"github.com/alibaba/sealer/utils"

	"testing"
)

func TestAssemblyIPList(t *testing.T) {
	tests := []struct {
		name    string
		args    *common.RunArgs
		wantErr bool
	}{
		{
			"baseData",
			&common.RunArgs{
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
			&common.RunArgs{
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
			&common.RunArgs{
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
			&common.RunArgs{
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
			&common.RunArgs{
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
			if err := utils.AssemblyIPList(&tt.args.Masters); (err != nil) != tt.wantErr {
				logger.Error("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
			}
			logger.Info("masters : %v , nodes : %v", &tt.args.Masters, &tt.args.Nodes)
		})
	}
}
