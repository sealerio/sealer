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

package env

import (
	"testing"
)

func Test_processor_WrapperShell(t *testing.T) {
	type args struct {
		wrapperData map[string]string
		shell       string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			"test WrapperShell ",
			args{
				wrapperData: map[string]string{
					"foo": "bar",
					"IP":  "127.0.0.1",
				},
				shell: "hostname",
			},
			"export IP=\"127.0.0.1\"; export foo=\"bar\"; hostname",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WrapperShell(tt.args.shell, tt.args.wrapperData); got != tt.want {
				t.Errorf("WrapperShell() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_processor_RenderAll(t *testing.T) {
	type args struct {
		renderData map[string]string
		dir        string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"test render dir",
			args{
				renderData: map[string]string{
					"PodCIDR": "100.64.0.0/10",
					"SvcCIDR": "10.96.0.0/16",
				},
				dir: "test/template",
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := RenderTemplate(tt.args.dir, tt.args.renderData); (err != nil) != tt.wantErr {
				t.Errorf("RenderAll() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
