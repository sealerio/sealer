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

package application

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	v2 "github.com/sealerio/sealer/types/api/v2"
)

func Test_mergeProcessor_Process(t *testing.T) {
	tests := []struct {
		source   string
		patch    string
		expected string
		wantErr  bool
	}{
		{
			source:   "a: b\nb: c",
			patch:    "b: d",
			expected: "a: b\nb: d\n",
			wantErr:  false,
		},
		{
			source:   "a: b\n## ---\nb: c",
			patch:    "b: d",
			expected: "a: b\nb: d\n",
			wantErr:  false,
		},
		{
			source:   "a: b\n---\nb: c",
			patch:    "b: d",
			expected: "a: b\nb: d\n\n---\nb: d\n",
			wantErr:  false,
		},
	}
	// prepare a tmp dir to write test files
	appRoot, err := os.MkdirTemp("", "sealer-unit-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.RemoveAll(appRoot)
	}()

	testFile := filepath.Join(appRoot, "test.yaml")
	for _, tt := range tests {
		// prepare source file
		f, err := os.Create(testFile)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString(tt.source)
		if err != nil {
			t.Fatal(err)
		}

		r := mergeProcessor{
			AppFile: v2.AppFile{
				Path:     "test.yaml",
				Strategy: v2.MergeStrategy,
				Data:     tt.patch,
			},
		}

		if err := r.Process(appRoot); err != nil && !tt.wantErr {
			t.Errorf("mergeProcessor.Process() error = %v", err)
		} else {
			output, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(string(output), tt.expected); diff != "" {
				t.Errorf("test failed expected=%s; output=%s; diff=%s", tt.expected, string(output), diff)
			}
		}
	}
}
