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
	"net"
	"testing"

	"github.com/sirupsen/logrus"
)

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
			if res := removeIPList(tt.clusterIPList, tt.toBeDeletedIPList); (res != nil) != tt.wantErr {
				fmt.Println(res)
			}
			logrus.Error("is empty")
		})
	}
}
