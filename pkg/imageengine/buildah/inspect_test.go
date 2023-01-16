// Copyright © 2023 Alibaba Group Holding Ltd.
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

package buildah

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/sealerio/sealer/build/kubefile/command"
)

func Test_handleLabelOutput(t *testing.T) {
	type args struct {
		labels map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "",
			args: args{
				labels: map[string]string{
					fmt.Sprintf("%s%s", command.LabelKubeCNIPrefix, "calico"):      "true",
					fmt.Sprintf("%s%s", command.LabelKubeCNIPrefix, "flannel"):     "true",
					fmt.Sprintf("%s%s", command.LabelKubeCSIPrefix, "alibaba-csi"): "true",
					"key1": "value1",
				},
			},
			want: map[string]string{
				command.LabelSupportedKubeCNIAlpha: `["calico","flannel"]`,
				command.LabelSupportedKubeCSIAlpha: `["alibaba-csi"]`,
				"key1":                             "value1",
			},
		},
		{
			name: "",
			args: args{
				labels: map[string]string{},
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := handleImageLabelOutput(tt.args.labels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handleLabelOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}
