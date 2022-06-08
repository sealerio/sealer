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

package os

/* func TestReadAll(t *testing.T) {
	type args struct {
		fileName string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		// TODO: Add test cases.
		{
			"test from ./test/file/123.txt",
			args{fileName: "./test/file/123.txt"},
			[]byte("123456"),
		},
		{
			"test from ./test/file/abc.txt",
			args{fileName: "./test/file/abc.txt"},
			[]byte("a\r\nb\r\nc\r\nd"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := NewFileReader(tt.args.fileName).ReadAll(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadAll() = %v, want %v", got, tt.want)
			}
		})
	}
}*/
