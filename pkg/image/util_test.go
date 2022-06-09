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

package image

import (
	"testing"
)

func Test_ConvertToHostname(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"url with http prefix",
			args{url: "http://registry.cn-qingdao.aliyuncs.com"},
			"registry.cn-qingdao.aliyuncs.com",
		},
		{
			"url with https prefix",
			args{url: "https://registry.cn-qingdao.aliyuncs.com"},
			"registry.cn-qingdao.aliyuncs.com",
		},
		{
			"url without prefix",
			args{url: "registry.cn-qingdao.aliyuncs.com"},
			"registry.cn-qingdao.aliyuncs.com",
		},
		{
			"url with custom port",
			args{url: "registry.cn-qingdao.aliyuncs.com:5000"},
			"registry.cn-qingdao.aliyuncs.com:5000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToHostname(tt.args.url); got != tt.want {
				t.Errorf("ConvertToHostname() = %v, want %v", got, tt.want)
			}
		})
	}
}
