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

package layerutils

import (
	"reflect"
	"testing"
)

func Test_decodeImages(t *testing.T) {
	body := `
          image: cn-app-integration:v1.0.0
          image: registry.cn-shanghai.aliyuncs.com/cnip/cn-app-integration:v1.0.0
          imagePullPolicy: Always
          image: cn-app-integration:v1.0.0
		  # image: cn-app-integration:v1.0.0
          name: cn-app-demo`
	type args struct {
		body string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			"test get image from yaml",
			args{body},
			[]string{"cn-app-integration:v1.0.0", "registry.cn-shanghai.aliyuncs.com/cnip/cn-app-integration:v1.0.0", "cn-app-integration:v1.0.0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DecodeImages(tt.args.body); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("decodeImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_decodeLine(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				line: "image: registry.cn-shanghai.aliyuncs.com/sealerio/foo:v1.0.0",
			},
			want: "registry.cn-shanghai.aliyuncs.com/sealerio/foo:v1.0.0",
		},
		{
			name: "withBlank",
			args: args{
				line: "    image: registry.cn-shanghai.aliyuncs.com/sealerio/foo:v1.0.0",
			},
			want: "registry.cn-shanghai.aliyuncs.com/sealerio/foo:v1.0.0",
		},
		{
			name: "withPrefix",
			args: args{
				line: "vmagent-image: \"ack-agility-registry.cn-shanghai.cr.aliyuncs.com/sealerio/foo:v1.70.0\"",
			},
			want: "ack-agility-registry.cn-shanghai.cr.aliyuncs.com/sealerio/foo:v1.70.0",
		},
		{
			name: "inArray",
			args: args{
				line: "- image: \"ack-agility-registry.cn-shanghai.cr.aliyuncs.com/sealerio/foo:v1.70.0\"",
			},
			want: "ack-agility-registry.cn-shanghai.cr.aliyuncs.com/sealerio/foo:v1.70.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := decodeLine(tt.args.line); got != tt.want {
				t.Errorf("decodeLine() = %v, want %v", got, tt.want)
			}
		})
	}
}
