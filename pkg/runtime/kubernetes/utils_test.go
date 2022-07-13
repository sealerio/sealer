// Copyright © 2022 Alibaba Group Holding Ltd.
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

package kubernetes

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestVersionCompare(t *testing.T) {
	tests := []struct {
		oldVersion string
		newVersion string
		name       string
		expectRes  bool
	}{
		{
			"v1.24.1",
			"v1.19.8",
			"test: version v1<v2",
			true,
		},
		{
			"v1.19.8",
			"v1.20.4",
			"test: version v1>v2",
			false,
		},
		{
			"v1.24.1",
			"v1.24.1",
			"test: version v1=v2",
			true,
		},
		{
			"",
			"",
			"test: version field is blank",
			false,
		},
		{
			"1.24.x",
			"1.x.8",
			"test: version field not legal",
			false,
		},
		{
			"-v1.24.1",
			"-v1.19.8",
			"test: version is wrong",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := VersionCompare(tt.oldVersion, tt.newVersion); res != tt.expectRes {
				logrus.Errorf("oldVersion: %s, newVersion: %s. compare should be: %v, but return: %v", tt.oldVersion, tt.newVersion, tt.expectRes, res)
			}
		})
	}
}
