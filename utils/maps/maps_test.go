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

package maps

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Merge(t *testing.T) {
	src := map[string]string{
		"key1":    "src1",
		"key2":    "src2",
		"key3":    "src3",
		"src-key": "src-value",
	}

	dst := map[string]string{
		"key1":    "v1",
		"key2":    "v2",
		"key3":    "v3",
		"dst-key": "dst-value",
	}

	result := map[string]string{
		"key1":    "v1",
		"key2":    "v2",
		"key3":    "v3",
		"dst-key": "dst-value",
		"src-key": "src-value",
	}

	nilDst := make(map[string]string)

	type args struct {
		src    map[string]string
		dst    map[string]string
		wanted map[string]string
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			name: "nil dst want get src as result",
			args: args{
				src:    src,
				dst:    nilDst,
				wanted: src,
			},
		},
		{
			name: "not nil dst want get overwriting dst with src as result",
			args: args{
				src:    src,
				dst:    dst,
				wanted: result,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.args.wanted, Merge(tt.args.dst, tt.args.src))
		})
	}
}

func Test_Copy(t *testing.T) {
	src := map[string]string{
		"key1":    "v1",
		"key2":    "v2",
		"key3":    "v3",
		"dst-key": "dst-value",
		"src-key": "src-value",
	}

	result := map[string]string{
		"key1":    "v1",
		"key2":    "v2",
		"key3":    "v3",
		"dst-key": "dst-value",
		"src-key": "src-value",
	}

	type args struct {
		src    map[string]string
		wanted map[string]string
	}

	var tests = []struct {
		name string
		args args
	}{
		{
			name: "nil src want get nil result",
			args: args{
				src:    nil,
				wanted: nil,
			},
		},
		{
			name: "not nil src want same src as result",
			args: args{
				src:    src,
				wanted: result,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.args.wanted, Copy(tt.args.src))
		})
	}
}
