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
	"testing"
)

func TestListImages(t *testing.T) {
	type args struct {
		clusterName string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			"test list charts images",
			args{"my_cluster"},
			[]string{"docker.elastic.co/elasticsearch/elasticsearch:7.13.2", "traefik:2.4.9"},
			false,
		},
	}
	charts, _ := NewCharts()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := charts.ListImages(tt.args.clusterName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ListImages() got = %v, want %v", got, tt.want)
			}
		})
	}
}
