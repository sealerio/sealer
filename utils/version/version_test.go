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

package version

import (
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name         string
		givenVersion Version
		oldVersion   Version
		wantRes      bool
	}{
		{
			name:         "test v > v1",
			givenVersion: "v1.20.4",
			oldVersion:   "v1.19.8",
			wantRes:      true,
		},
		{
			name:         "test v = v1",
			givenVersion: "v1.19.8",
			oldVersion:   "v1.19.8",
			wantRes:      true,
		},
		{
			name:         "test v < v1",
			givenVersion: "v1.19.8",
			oldVersion:   "v1.20.4",
			wantRes:      false,
		},
		{
			name:         "test1 old Version illegal",
			givenVersion: "v1.19.8",
			oldVersion:   "",
			wantRes:      false,
		},
		{
			name:         "test2 give Version illegal",
			givenVersion: "",
			oldVersion:   "v1.19.8",
			wantRes:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := tt.givenVersion
			res, err := v.GreaterThan(tt.oldVersion)
			if err != nil {
				logrus.Errorf("compare kubernetes version failed: %s", err)
			}
			if !reflect.DeepEqual(res, tt.wantRes) {
				t.Errorf("Version compare failed! result: %v, want: %v", res, tt.wantRes)
			}
		})
	}
}
