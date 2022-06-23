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
	"testing"

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
			resultIPStr, err := AssemblyIPList(tt.ipStr)
			if err != nil && tt.wantErr == false {
				t.Errorf("input ipStr(%s), found non-nil error(%v), but expect nil error. returned ipStr(%s)", tt.ipStr, err, resultIPStr)
			}
			if err == nil && tt.wantErr == true {
				t.Errorf("input ipStr(%s), found nil error, but expect non-nil error,returned ipStr(%s)", tt.ipStr, resultIPStr)
			}
		})
	}
}
