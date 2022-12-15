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

package net

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sirupsen/logrus"
)

func TestAssemblyIPList(t *testing.T) {
	tests := []struct {
		name    string
		ipStr   string
		wantErr bool
	}{
		{
			name:    "baseData1",
			ipStr:   "10.110.101.1-10.110.101.5",
			wantErr: false,
		},
		/*{
			name:    "baseData2",
			ipStr:   "0.0.0.0-0.0.0.1",
			wantErr: false,
		},*/
		{
			name:    "errorData",
			ipStr:   "10.110.101.10-10.110.101.5",
			wantErr: true,
		},
		{
			name:    "errorData2",
			ipStr:   "10.110.101.10-10.110.101.5-10.110.101.55",
			wantErr: true,
		},
		{
			name:    "errorData3",
			ipStr:   "-10.110.101.",
			wantErr: true,
		},
		{
			name:    "errorData4",
			ipStr:   "10.110.101.1-",
			wantErr: true,
		},
		{
			name:    "errorData5",
			ipStr:   "a-b",
			wantErr: true,
		},
		{
			name:    "errorData6",
			ipStr:   "-b",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logrus.Infof("start to test case %s", tt.name)
			resultIPStr, err := TransferToIPList(tt.ipStr)
			if err != nil && tt.wantErr == false {
				t.Errorf("input ipStr(%s), found non-nil error(%v), but expect nil error. returned ipStr(%s)", tt.ipStr, err, resultIPStr)
			}
			if err == nil && tt.wantErr == true {
				t.Errorf("input ipStr(%s), found nil error, but expect non-nil error,returned ipStr(%s)", tt.ipStr, resultIPStr)
			}
		})
	}
}

func TestIPStrsToIPs(t *testing.T) {
	tests := []struct {
		name        string
		inputIPStrs []string
		wantedIPs   []net.IP
	}{
		{
			name:        "baseData1",
			inputIPStrs: []string{"10.110.101.1"},
			wantedIPs:   []net.IP{net.ParseIP("10.110.101.1")},
		},
		{
			name:        "baseData2",
			inputIPStrs: []string{"10.110.101.1", "sdfghjkl"},
			wantedIPs:   []net.IP{net.ParseIP("10.110.101.1"), nil},
		},
		{
			name:        "baseData2",
			inputIPStrs: []string{"10.110.101.1", "10.110.101.100"},
			wantedIPs:   []net.IP{net.ParseIP("10.110.101.1"), net.ParseIP("10.110.101.100")},
		},
		{
			name:        "empty input of nil",
			inputIPStrs: nil,
			wantedIPs:   nil,
		},
		{
			name:        "non-empty input with empty string",
			inputIPStrs: []string{""},
			wantedIPs:   []net.IP{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logrus.Infof("start to test case %s", tt.name)
			ips := IPStrsToIPs(tt.inputIPStrs)
			if !equalNetIPs(ips, tt.wantedIPs) {
				t.Errorf("wanted ips is (%s), but got (%s)", tt.wantedIPs, ips)
			}
		})
	}
}

func TestIPsToIPStrs(t *testing.T) {
	tests := []struct {
		name         string
		inputIPs     []net.IP
		wantedIPStrs []string
	}{
		{
			name:         "baseData1",
			inputIPs:     []net.IP{net.ParseIP("10.110.101.1")},
			wantedIPStrs: []string{"10.110.101.1"},
		},
		{
			name:         "baseData2",
			inputIPs:     []net.IP{net.ParseIP("10.110.101.1"), net.ParseIP("10.110.101.2")},
			wantedIPStrs: []string{"10.110.101.1", "10.110.101.2"},
		},
		{
			name:         "baseData3",
			inputIPs:     []net.IP{net.ParseIP("10.110.101.1"), net.ParseIP("10.110.101.653")},
			wantedIPStrs: []string{"10.110.101.1", "<nil>"},
		},
		{
			name:         "empty input of nil",
			inputIPs:     nil,
			wantedIPStrs: nil,
		},
		{
			name:         "non-empty input with empty string",
			inputIPs:     []net.IP{nil},
			wantedIPStrs: []string{"<nil>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logrus.Infof("start to test case %s", tt.name)
			ipStrs := IPsToIPStrs(tt.inputIPs)
			if !equalIPStrs(ipStrs, tt.wantedIPStrs) {
				t.Errorf("wanted IP strings is (%s), but got (%s)", tt.wantedIPStrs, ipStrs)
			}
		})
	}
}

func Test_returnFilteredIPList(t *testing.T) {
	tests := []struct {
		name              string
		clusterIPList     []net.IP
		toBeDeletedIPList []net.IP
		IPListExpected    []net.IP
	}{
		{
			"test",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
		},
		{
			"test1",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
		},
		{
			"test2",
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.4")},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3")},
		},
		{
			"test3",
			[]net.IP{},
			[]net.IP{net.ParseIP("10.10.10.1"), net.ParseIP("10.10.10.2"), net.ParseIP("10.10.10.3"), net.ParseIP("10.10.10.4")},
			[]net.IP{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := RemoveIPs(tt.clusterIPList, tt.toBeDeletedIPList); res != nil {
				assert.Equal(t, tt.IPListExpected, res)
			}
		})
	}
}

func equalNetIPs(a, b []net.IP) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if !a[index].Equal(b[index]) {
			return false
		}
	}
	return true
}

func equalIPStrs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}
