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

package apply

import (
	"testing"

	"github.com/alibaba/sealer/utils"
)

func TestAppendFile(t *testing.T) {
	type args struct {
		content  string
		fileName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"add hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/hosts1",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.AppendFile(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("AppendFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRemoveFileContent(t *testing.T) {
	type args struct {
		fileName string
		content  string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			"delete hosts",
			args{
				content:  "127.0.0.1 localhost",
				fileName: "./test/hosts1",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := utils.RemoveFileContent(tt.args.fileName, tt.args.content); (err != nil) != tt.wantErr {
				t.Errorf("RemoveFileContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
