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
	"net"
	"testing"

	"github.com/sealerio/sealer/common"
	"github.com/sirupsen/logrus"
)

func TestNewCleanApplierFromArgs(t *testing.T) {
	tests := []struct {
		cFile   string
		cArgs   *Args
		name    string
		flag    string
		wantErr bool
	}{
		{
			"Clusterfile",
			&Args{
				Masters: "10.110.101.1-10.110.101.5",
				Nodes:   "10.110.101.1-10.110.101.5",
			},
			"test1",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "10.110.101.1,10.110.101.2",
				Nodes:   "10.110.101.1,10.110.101.5",
			},
			"test2",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "2",
				Nodes:   "1",
			},
			"test3",
			common.DeleteSubCmd,
			false,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "-10.110.101.2",
				Nodes:   "10.110.101.2-",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "-10.110.101.2",
				Nodes:   "10.110.101.2-",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "b-a",
				Nodes:   "a-b",
			},
			"test4",
			common.DeleteSubCmd,
			true,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "10.110.101.1,10.110.101.2",
			},
			"join only has master",
			common.JoinSubCmd,
			false,
		},

		{
			"Clusterfile",
			&Args{
				Nodes: "10.110.101.1,10.110.101.2",
			},
			"join only has node",
			common.JoinSubCmd,
			false,
		},
		{
			"Clusterfile",
			&Args{
				Masters: "10.110.101.1,10.110.101.2",
				Nodes:   "10.110.101.1,10.110.101.2",
			},
			"join master and node at the same time",
			common.JoinSubCmd,
			false,
		},
		{
			"Clusterfile",
			&Args{},
			"joined master and node are both empty",
			common.JoinSubCmd,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if applier, err := NewScaleApplierFromArgs(tt.cFile, tt.cArgs, tt.flag); (err != nil) != tt.wantErr {
				logrus.Errorf("masters : %v , nodes : %v , applier : %v", &tt.cArgs.Masters, &tt.cArgs.Nodes, applier)
			}
			logrus.Infof("masters : %v , nodes : %v", &tt.cArgs.Masters, &tt.cArgs.Nodes)
		})
	}
}

func Test_returnFilteredIPList(t *testing.T) {
	tests := []struct {
		name              string
		clusterIPList     []net.IP
		toBeDeletedIPList []net.IP
		wantErr           bool
	}{
		{
			"test",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			false,
		},
		{
			"test1",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
			false,
		},
		{
			"test2",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.4")},
			false,
		},
		{
			"test3",
			[]net.IP{},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := returnFilteredIPList(tt.clusterIPList, tt.toBeDeletedIPList); (res != nil) != tt.wantErr {
				fmt.Println(res)
			}
			logrus.Error("is empty")
		})
	}
}
