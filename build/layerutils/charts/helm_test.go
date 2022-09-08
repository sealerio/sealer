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

package charts

import (
	"reflect"
	"sort"
	"testing"
)

func TestGetImageList(t *testing.T) {
	type args struct {
		chartPath string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"test get image list in chart",
			args{"./testcharts/apps"},
			[]string{"nginx:apps_release", "nginx:app1_release", "nginx:app2_test"},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			images, err := GetImageList(tt.args.chartPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetImageList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Strings(images)
			sort.Strings(tt.want)
			if !reflect.DeepEqual(images, tt.want) {
				t.Errorf("GetImageList() error get %v, want %v", images, tt.want)
				return
			}
		})
	}
}
