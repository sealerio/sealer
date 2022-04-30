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
	"fmt"
	"testing"

	"github.com/sealerio/sealer/common"
	"github.com/sealerio/sealer/logger"
)

func TestNewCleanApplierFromArgs(t *testing.T) {
	tests := []struct {
		cFile   string
		cArgs   *common.RunArgs
		name    string
		flag    string
		wantErr bool
	}{
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "10.110.101.1-10.110.101.5",
				Nodes:   "10.110.101.1-10.110.101.5",
			},
			"test1",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "10.110.101.1,10.110.101.2",
				Nodes:   "10.110.101.1,10.110.101.5",
			},
			"test2",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "2",
				Nodes:   "1",
			},
			"test3",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "-10.110.101.2",
				Nodes:   "10.110.101.2-",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "-10.110.101.2",
				Nodes:   "10.110.101.2-",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
		{
			"Clusterfile",
			&common.RunArgs{
				Masters: "b-a",
				Nodes:   "a-b",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if applier, err := NewScaleApplierFromArgs(tt.cFile, tt.cArgs, tt.flag); (err != nil) != tt.wantErr {
				logger.Error("masters : %v , nodes : %v , applier : %v", &tt.cArgs.Masters, &tt.cArgs.Nodes, applier)
			}
			logger.Info("masters : %v , nodes : %v", &tt.cArgs.Masters, &tt.cArgs.Nodes)
		})
	}
}

func Test_returnFilteredIPList(t *testing.T) {
	tests := []struct {
		name              string
		clusterIPList     []string
		toBeDeletedIPList []string
		wantErr           bool
	}{
		{
			"test",
			[]string{"10.10.10.1", "10.10.10.2", "10.10.10.3", "10.10.10.4"},
			[]string{"10.10.10.1", "10.10.10.2", "10.10.10.3", "10.10.10.4"},
			false,
		},
		{
			"test1",
			[]string{"10.10.10.1", "10.10.10.2", "10.10.10.3", "10.10.10.4"},
			[]string{},
			false,
		},
		{
			"test2",
			[]string{"10.10.10.1", "10.10.10.2", "10.10.10.3", "10.10.10.4"},
			[]string{"10.10.10.4"},
			false,
		},
		{
			"test3",
			[]string{},
			[]string{"10.10.10.1", "10.10.10.2", "10.10.10.3", "10.10.10.4"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := returnFilteredIPList(tt.clusterIPList, tt.toBeDeletedIPList); (res != nil) != tt.wantErr {
				fmt.Println(res)
			}
			logger.Error("is empty")
		})
	}
}
